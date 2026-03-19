package repositories

import (
	"backend/models/domains"

	"gorm.io/gorm"
)

type OrderRepo interface {
	Create(db *gorm.DB, schema string, order domains.Order) (*domains.Order, error)
	GetAll(db *gorm.DB, schema string, pagination domains.Pagination) ([]domains.Order, int, error)
	GetByID(db *gorm.DB, schema string, id int) (*domains.Order, error)
	UpdateStatus(db *gorm.DB, schema string, id int, status domains.OrderStatus) error
	CreateOrderProducts(db *gorm.DB, schema string, products []domains.OrderProduct) error
}
