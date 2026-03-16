package domains

import (
	"time"

	"github.com/google/uuid"
)

type Plan struct {
	ID        uuid.UUID `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	Name      string    `gorm:"column:name;not null"`
	IsMonth   bool      `gorm:"column:is_month;not null;default:true"`
	Duration  int       `gorm:"column:duration;not null"`
	Price     float64   `gorm:"column:price;type:numeric(15,2);not null;default:0"`
	IsActive  bool      `gorm:"column:is_active;not null;default:true"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Plan) TableName() string {
	return "plan"
}
