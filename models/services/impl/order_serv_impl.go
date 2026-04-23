package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	reqOrder "backend/models/requests/order"
	resOrder "backend/models/responses/order"
	"backend/models/responses/pagination"
	"errors"
	"fmt"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderServImpl struct {
	Db                  *gorm.DB
	JwtKey              string
	Validator           *validator.Validate
	UserRepo            repositories.UsersRepo
	CustomerRepo        repositories.CustomerRepo
	OrderRepo           repositories.OrderRepo
	ProductRepo         repositories.ProductRepo
	DeliverySettingRepo repositories.DeliverySettingRepo
}

func NewOrderServImpl(
	db *gorm.DB,
	jwtKey string,
	validator *validator.Validate,
	userRepo repositories.UsersRepo,
	customerRepo repositories.CustomerRepo,
	orderRepo repositories.OrderRepo,
	productRepo repositories.ProductRepo,
	deliverySettingRepo repositories.DeliverySettingRepo,
) *OrderServImpl {
	return &OrderServImpl{
		Db:                  db,
		JwtKey:              jwtKey,
		Validator:           validator,
		UserRepo:            userRepo,
		CustomerRepo:        customerRepo,
		OrderRepo:           orderRepo,
		ProductRepo:         productRepo,
		DeliverySettingRepo: deliverySettingRepo,
	}
}

func (serv *OrderServImpl) checkRole(accessToken string) error {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Admin", "Client"})
	if err != nil || !ok {
		return fmt.Errorf("access denied")
	}
	return nil
}

func (serv *OrderServImpl) getSchema(clientID uuid.UUID) (string, error) {
	return helpers.GetSchema(serv.Db, serv.UserRepo, clientID)
}

func (serv *OrderServImpl) getDeliveryName(schema, subGroupName string) string {
	settings, err := serv.DeliverySettingRepo.GetBySubGroupName(serv.Db, schema, subGroupName)
	if err != nil || len(settings) == 0 {
		return ""
	}
	deliveries := domains.ToDeliverySetting(settings)
	if len(deliveries) == 0 {
		return ""
	}
	return deliveries[0].Name
}

func (serv *OrderServImpl) buildProductDetails(schema string, orderProducts []domains.OrderProduct) []resOrder.ProductDetailResponse {
	var result []resOrder.ProductDetailResponse
	for _, op := range orderProducts {
		productID, err := uuid.Parse(op.ProductID)
		productName := ""
		price := op.TotalPrice
		if err == nil {
			product, err := serv.ProductRepo.GetByID(serv.Db, schema, productID)
			if err == nil {
				productName = product.Name
				price = product.Price
			}
		}
		result = append(result, resOrder.ProductDetailResponse{
			ID:          op.ID,
			ProductID:   op.ProductID,
			ProductName: productName,
			Price:       price,
			Quantity:    op.Quantity,
			TotalPrice:  op.TotalPrice,
		})
	}
	return result
}

func (serv *OrderServImpl) Create(accessToken string, clientID uuid.UUID, request reqOrder.CreateOrderRequest) (*resOrder.DetailResponse, error) {
	if err := serv.checkRole(accessToken); err != nil {
		return nil, err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(clientID)
	if err != nil {
		return nil, err
	}

	tx := serv.Db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to start transaction")
	}

	// Cek customer by phone — kalau ada pakai yang ada, kalau tidak ada buat baru
	customer, err := serv.CustomerRepo.GetByPhone(tx, schema, request.PhoneCountryCode, request.PhoneNumber)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			customer, err = serv.CustomerRepo.Create(tx, schema, domains.Customer{
				Name:             request.CustomerName,
				PhoneCountryCode: &request.PhoneCountryCode,
				PhoneNumber:      &request.PhoneNumber,
				AccountType:      request.AccountType,
			})
			if err != nil {
				tx.Rollback()
				log.Printf("[CustomerRepo].Create error: %v", err)
				return nil, fmt.Errorf("failed to create customer")
			}
		} else {
			tx.Rollback()
			return nil, fmt.Errorf("failed to check customer")
		}
	}

	// Hitung total price
	var totalPrice float64
	var orderProducts []domains.OrderProduct

	for _, p := range request.Products {
		productID, err := uuid.Parse(p.ProductID)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("invalid product_id: %s", p.ProductID)
		}

		product, err := serv.ProductRepo.GetByID(tx, schema, productID)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("product %s not found", p.ProductID)
		}

		itemTotal := product.Price * float64(p.Quantity)
		totalPrice += itemTotal
		orderProducts = append(orderProducts, domains.OrderProduct{
			ProductID:  p.ProductID,
			Quantity:   p.Quantity,
			TotalPrice: itemTotal,
		})
	}

	order, err := serv.OrderRepo.Create(tx, schema, domains.Order{
		CustomerID:           customer.ID,
		TotalPrice:           totalPrice,
		Status:               domains.OrderStatusPending,
		DeliverySubGroupName: request.DeliverySubGroupName,
		StreetAddress:        request.StreetAddress,
		PostalCode:           request.PostalCode,
	})
	if err != nil {
		tx.Rollback()
		log.Printf("[OrderRepo].Create error: %v", err)
		return nil, fmt.Errorf("failed to create order")
	}

	for i := range orderProducts {
		orderProducts[i].OrderID = order.ID
	}

	if err := serv.OrderRepo.CreateOrderProducts(tx, schema, orderProducts); err != nil {
		tx.Rollback()
		log.Printf("[OrderRepo].CreateOrderProducts error: %v", err)
		return nil, fmt.Errorf("failed to create order products")
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to commit transaction")
	}

	fullOrder, err := serv.OrderRepo.GetByID(serv.Db, schema, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order")
	}

	deliveryName := serv.getDeliveryName(schema, fullOrder.DeliverySubGroupName)
	productDetails := serv.buildProductDetails(schema, fullOrder.Products)
	response := resOrder.ToDetailResponse(*fullOrder, deliveryName, productDetails)
	return &response, nil
}

func (serv *OrderServImpl) GetAll(accessToken string, clientID uuid.UUID, pg domains.Pagination) (*pagination.Response, error) {
	if err := serv.checkRole(accessToken); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(clientID)
	if err != nil {
		return nil, err
	}

	orders, total, err := serv.OrderRepo.GetAll(serv.Db, schema, pg)
	if err != nil {
		log.Printf("[OrderRepo].GetAll error: %v", err)
		return nil, fmt.Errorf("failed to get orders")
	}

	deliveryMap := map[string]string{}
	for _, o := range orders {
		if _, exists := deliveryMap[o.DeliverySubGroupName]; !exists {
			deliveryMap[o.DeliverySubGroupName] = serv.getDeliveryName(schema, o.DeliverySubGroupName)
		}
	}

	responses := resOrder.ToResponses(orders, deliveryMap)
	result := pagination.ToResponse(responses, total, pg.Page, pg.Limit)
	return &result, nil
}

func (serv *OrderServImpl) GetByID(accessToken string, clientID uuid.UUID, id int) (*resOrder.DetailResponse, error) {
	if err := serv.checkRole(accessToken); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(clientID)
	if err != nil {
		return nil, err
	}

	order, err := serv.OrderRepo.GetByID(serv.Db, schema, id)
	if err != nil {
		log.Printf("[OrderRepo].GetByID error: %v", err)
		return nil, fmt.Errorf("order not found")
	}

	deliveryName := serv.getDeliveryName(schema, order.DeliverySubGroupName)
	productDetails := serv.buildProductDetails(schema, order.Products)
	response := resOrder.ToDetailResponse(*order, deliveryName, productDetails)
	return &response, nil
}

func (serv *OrderServImpl) UpdateStatus(accessToken string, clientID uuid.UUID, id int, request reqOrder.UpdateOrderStatusRequest) error {
	if err := serv.checkRole(accessToken); err != nil {
		return err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return err
	}

	schema, err := serv.getSchema(clientID)
	if err != nil {
		return err
	}

	_, err = serv.OrderRepo.GetByID(serv.Db, schema, id)
	if err != nil {
		return fmt.Errorf("order not found")
	}

	if err := serv.OrderRepo.UpdateStatus(serv.Db, schema, id, request.Status); err != nil {
		log.Printf("[OrderRepo].UpdateStatus error: %v", err)
		return fmt.Errorf("failed to update order status")
	}

	return nil
}
