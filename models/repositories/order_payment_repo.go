package repositories

import (
	"backend/models/domains"

	"gorm.io/gorm"
)

type OrderPaymentRepo interface {
	Create(db *gorm.DB, schema string, payment domains.OrderPayment) (*domains.OrderPayment, error)
	GetAll(db *gorm.DB, schema string, pagination domains.Pagination) ([]domains.OrderPayment, int, error)
	GetByOrderID(db *gorm.DB, schema string, orderID int) (*domains.OrderPayment, error)
	UpdateStatus(db *gorm.DB, schema string, orderID int, status domains.PaymentStatus) error
}
