package domains

import (
	"time"

	"github.com/google/uuid"
)

type BusinessProfile struct {
	ID           uuid.UUID `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantId     uuid.UUID `gorm:"column:tenant_id;not null;type:uuid"`
	BusinessName string    `gorm:"column:business_name"`
	Address      string    `gorm:"column:address"`
	Phone        string    `gorm:"column:phone"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relations
	Tenant Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
}

func (BusinessProfile) TableName() string {
	return "business_profile"
}
