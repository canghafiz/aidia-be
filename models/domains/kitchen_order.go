package domains

import (
	"time"

	"github.com/google/uuid"
)

type KitchenStatus string

const (
	KitchenStatusNewOrder KitchenStatus = "new_order"
	KitchenStatusCooking  KitchenStatus = "cooking"
	KitchenStatusPacking  KitchenStatus = "packing"
	KitchenStatusReady    KitchenStatus = "ready"
)

type KitchenOrder struct {
	ID        uuid.UUID     `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	OrderID   int           `gorm:"column:order_id;not null;uniqueIndex"`
	Status    KitchenStatus `gorm:"column:status;not null;default:'new_order'"`
	CreatedAt time.Time     `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time     `gorm:"column:updated_at;autoUpdateTime"`

	// Relasi — load manual
	Order        *Order        `gorm:"-"`
	OrderPayment *OrderPayment `gorm:"-"`
}
