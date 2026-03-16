package domains

import (
	"time"

	"github.com/google/uuid"
)

type ProductImage struct {
	ID        uuid.UUID `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	ProductID uuid.UUID `gorm:"column:product_id;not null"`
	Image     string    `gorm:"column:image;not null"`
	IsActive  bool      `gorm:"column:is_active;not null;default:true"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}
