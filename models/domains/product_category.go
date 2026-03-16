package domains

import (
	"time"

	"github.com/google/uuid"
)

type ProductCategory struct {
	ID          uuid.UUID `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	Name        string    `gorm:"column:name;not null"`
	IsVisible   bool      `gorm:"column:is_visible;not null;default:true"`
	Description *string   `gorm:"column:description;default:null"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`

	Products []Product `gorm:"-"`
}
