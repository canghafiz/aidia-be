package domains

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentStatusUnpaid            PaymentStatus = "Unpaid"
	PaymentStatusConfirmingPayment PaymentStatus = "Confirming_Payment"
	PaymentStatusPaid              PaymentStatus = "Paid"
	PaymentStatusRefunded          PaymentStatus = "Refunded"
	PaymentStatusVoided            PaymentStatus = "Voided"
)

type OrderPayment struct {
	ID            uuid.UUID     `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	OrderID       int           `gorm:"column:order_id;not null"`
	PaymentStatus PaymentStatus `gorm:"column:payment_status;not null;default:'Unpaid'"`
	PaymentMethod string        `gorm:"column:payment_method;not null;default:'stripe'"`
	TotalPrice    float64       `gorm:"column:total_price;not null;default:0"`
	ExpireAt      time.Time     `gorm:"column:expire_at;not null"`
	CreatedAt     time.Time     `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time     `gorm:"column:updated_at;autoUpdateTime"`
}
