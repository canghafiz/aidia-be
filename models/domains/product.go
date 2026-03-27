package domains

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID            uuid.UUID `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	Name          string    `gorm:"column:name;not null"`
	Weight        float64   `gorm:"column:weight;not null;default:0"`
	Price         float64   `gorm:"column:price;not null;default:0"`
	OriginalPrice float64   `gorm:"column:original_price;not null;default:0"`
	Description   *string   `gorm:"column:description"`
	DeliveryID    uuid.UUID `gorm:"column:delivery_id;not null"`
	IsOutOfStock  bool      `gorm:"column:is_out_of_stock;not null;default:false"`
	IsActive      bool      `gorm:"column:is_active;not null;default:true"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`

	Images []ProductImage `gorm:"foreignKey:ProductID;references:ID"`
}
