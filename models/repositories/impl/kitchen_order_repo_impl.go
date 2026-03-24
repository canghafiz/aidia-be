package impl

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type KitchenOrderRepoImpl struct{}

func NewKitchenOrderRepoImpl() *KitchenOrderRepoImpl {
	return &KitchenOrderRepoImpl{}
}

func (repo *KitchenOrderRepoImpl) loadRelations(db *gorm.DB, schema string, orders []domains.KitchenOrder) ([]domains.KitchenOrder, error) {
	for i, ko := range orders {
		// Load order
		var order domains.Order
		if err := db.Raw(`SELECT * FROM `+schema+`.orders WHERE id = ?`, ko.OrderID).
			Scan(&order).Error; err == nil && order.ID != 0 {

			// Load customer
			var customer domains.Customer
			if err := db.Raw(`SELECT * FROM `+schema+`.customer WHERE id = ?`, order.CustomerID).
				Scan(&customer).Error; err == nil {
				order.Customer = &customer
			}

			// Load order products
			var products []domains.OrderProduct
			if err := db.Raw(`SELECT * FROM `+schema+`.order_products WHERE order_id = ?`, order.ID).
				Scan(&products).Error; err == nil {
				order.Products = products
			}

			orders[i].Order = &order
		}

		// Load order payment
		var payment domains.OrderPayment
		if err := db.Raw(`SELECT * FROM `+schema+`.order_payments WHERE order_id = ?`, ko.OrderID).
			Scan(&payment).Error; err == nil && payment.ID != uuid.Nil {
			orders[i].OrderPayment = &payment
		}
	}
	return orders, nil
}

func (repo *KitchenOrderRepoImpl) GetAll(db *gorm.DB, schema string) ([]domains.KitchenOrder, error) {
	var orders []domains.KitchenOrder
	if err := db.Raw(`
		SELECT * FROM ` + schema + `.kitchen_order
		WHERE status != 'completed'
		ORDER BY created_at ASC`).
		Scan(&orders).Error; err != nil {
		return nil, err
	}
	return repo.loadRelations(db, schema, orders)
}

func (repo *KitchenOrderRepoImpl) GetByStatus(db *gorm.DB, schema string, status domains.KitchenStatus) ([]domains.KitchenOrder, error) {
	var orders []domains.KitchenOrder
	if err := db.Raw(`
		SELECT * FROM `+schema+`.kitchen_order
		WHERE status = ?
		ORDER BY created_at ASC`, status).
		Scan(&orders).Error; err != nil {
		return nil, err
	}
	return repo.loadRelations(db, schema, orders)
}

func (repo *KitchenOrderRepoImpl) GetByID(db *gorm.DB, schema string, id uuid.UUID) (*domains.KitchenOrder, error) {
	var order domains.KitchenOrder
	if err := db.Raw(`SELECT * FROM `+schema+`.kitchen_order WHERE id = ?`, id).
		Scan(&order).Error; err != nil {
		return nil, err
	}
	if order.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	orders, err := repo.loadRelations(db, schema, []domains.KitchenOrder{order})
	if err != nil {
		return nil, err
	}
	return &orders[0], nil
}

func (repo *KitchenOrderRepoImpl) UpdateStatus(db *gorm.DB, schema string, id uuid.UUID, status domains.KitchenStatus) error {
	return db.Table(schema+".kitchen_order").
		Where("id = ?", id).
		Update("status", status).Error
}
