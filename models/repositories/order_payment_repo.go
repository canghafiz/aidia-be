package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderPaymentRepo interface {
	GetAll(db *gorm.DB, schema string, pagination domains.Pagination) ([]domains.OrderPayment, int, error)
	GetByID(db *gorm.DB, schema string, id uuid.UUID) (*domains.OrderPayment, error)
	GetByOrderID(db *gorm.DB, schema string, orderID int) (*domains.OrderPayment, error)
	UpdateStatus(db *gorm.DB, schema string, id uuid.UUID, status domains.PaymentStatus) error
}
