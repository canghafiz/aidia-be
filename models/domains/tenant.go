package domains

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	TenantID  uuid.UUID `gorm:"column:tenant_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"column:user_id;not null;uniqueIndex:uq_tenant_user_id;type:uuid"`
	Role      string    `gorm:"column:role;not null;default:owner"`
	IsActive  bool      `gorm:"column:is_active;not null;default:true"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relations
	User            Users            `gorm:"foreignKey:UserID;references:UserID"`
	BusinessProfile *BusinessProfile `gorm:"foreignKey:TenantID;references:TenantID"`
}

func (Tenant) TableName() string {
	return "tenant"
}
