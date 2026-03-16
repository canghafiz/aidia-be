package domains

import (
	"time"

	"github.com/google/uuid"
)

type Users struct {
	UserID       uuid.UUID `gorm:"column:user_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	Username     string    `gorm:"column:username;not null;uniqueIndex:uq_users_username"`
	Name         string    `gorm:"column:name;not null"`
	Email        string    `gorm:"column:email;not null;uniqueIndex:uq_users_email"`
	Password     string    `gorm:"column:password;not null"`
	Gender       string    `gorm:"column:gender"`
	TenantSchema *string   `gorm:"column:tenant_schema"`
	IsActive     bool      `gorm:"column:is_active"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relations
	Tenant             *Tenant              `gorm:"foreignKey:UserID;references:UserID"`
	UserRoles          []UserRoles          `gorm:"foreignKey:UserID;references:UserID"`
	TenantApprovalLogs []TenantApprovalLogs `gorm:"foreignKey:ActionBy;references:UserID"`
}

func (Users) TableName() string {
	return "users"
}
