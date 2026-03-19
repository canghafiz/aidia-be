package impl

import (
	"backend/models/domains"

	"gorm.io/gorm"
)

type OrderRepoImpl struct{}

func NewOrderRepoImpl() *OrderRepoImpl {
	return &OrderRepoImpl{}
}

func (repo *OrderRepoImpl) Create(db *gorm.DB, schema string, order domains.Order) (*domains.Order, error) {
	if err := db.Table(schema + ".orders").Create(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (repo *OrderRepoImpl) GetAll(db *gorm.DB, schema string, pagination domains.Pagination) ([]domains.Order, int, error) {
	var orders []domains.Order
	var total int64

	if err := db.Table(schema + ".orders").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Raw(`
		SELECT * FROM `+schema+`.orders
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, pagination.Limit, pagination.Offset()).
		Scan(&orders).Error; err != nil {
		return nil, 0, err
	}

	for i, o := range orders {
		var customer domains.Customer
		if err := db.Raw(`SELECT * FROM `+schema+`.customer WHERE id = ?`, o.CustomerID).
			Scan(&customer).Error; err == nil {
			orders[i].Customer = &customer
		}

		var products []domains.OrderProduct
		if err := db.Raw(`SELECT * FROM `+schema+`.order_products WHERE order_id = ?`, o.ID).
			Scan(&products).Error; err == nil {
			orders[i].Products = products
		}

		var payment domains.OrderPayment
		if err := db.Raw(`SELECT * FROM `+schema+`.order_payments WHERE order_id = ?`, o.ID).
			Scan(&payment).Error; err == nil {
			orders[i].Payment = &payment
		}
	}

	return orders, int(total), nil
}

func (repo *OrderRepoImpl) GetByID(db *gorm.DB, schema string, id int) (*domains.Order, error) {
	var order domains.Order
	if err := db.Raw(`SELECT * FROM `+schema+`.orders WHERE id = ?`, id).
		Scan(&order).Error; err != nil {
		return nil, err
	}
	if order.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var customer domains.Customer
	if err := db.Raw(`SELECT * FROM `+schema+`.customer WHERE id = ?`, order.CustomerID).
		Scan(&customer).Error; err == nil {
		order.Customer = &customer
	}

	var products []domains.OrderProduct
	if err := db.Raw(`SELECT * FROM `+schema+`.order_products WHERE order_id = ?`, id).
		Scan(&products).Error; err == nil {
		order.Products = products
	}

	var payment domains.OrderPayment
	if err := db.Raw(`SELECT * FROM `+schema+`.order_payments WHERE order_id = ?`, id).
		Scan(&payment).Error; err == nil {
		order.Payment = &payment
	}

	return &order, nil
}

func (repo *OrderRepoImpl) UpdateStatus(db *gorm.DB, schema string, id int, status domains.OrderStatus) error {
	return db.Table(schema+".orders").
		Where("id = ?", id).
		Update("status", status).Error
}

func (repo *OrderRepoImpl) CreateOrderProducts(db *gorm.DB, schema string, products []domains.OrderProduct) error {
	return db.Table(schema + ".order_products").Create(&products).Error
}
