package domains

import "time"

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "Pending"
	OrderStatusConfirmed OrderStatus = "Confirmed"
	OrderStatusCompleted OrderStatus = "Completed"
	OrderStatusCancelled OrderStatus = "Cancelled"
)

type Order struct {
	ID                   int         `gorm:"column:id;primaryKey;autoIncrement"`
	CustomerID           int         `gorm:"column:customer_id;not null"`
	TotalPrice           float64     `gorm:"column:total_price;not null;default:0"`
	Status               OrderStatus `gorm:"column:status;not null;default:'Pending'"`
	DeliverySubGroupName string      `gorm:"column:delivery_sub_group_name;not null"`
	StreetAddress        string      `gorm:"column:street_address;not null"`
	PostalCode           string      `gorm:"column:postal_code;not null"`
	CreatedAt            time.Time   `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt            time.Time   `gorm:"column:updated_at;autoUpdateTime"`

	Customer *Customer      `gorm:"-"`
	Products []OrderProduct `gorm:"-"`
	Payment  *OrderPayment  `gorm:"-"`
}

type OrderProduct struct {
	ID         int       `gorm:"column:id;primaryKey;autoIncrement"`
	OrderID    int       `gorm:"column:order_id;not null"`
	ProductID  string    `gorm:"column:product_id;not null"`
	Quantity   int       `gorm:"column:quantity;not null;default:1"`
	TotalPrice float64   `gorm:"column:total_price;not null;default:0"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}
