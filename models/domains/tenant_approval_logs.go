package domains

import (
	"time"

	"github.com/google/uuid"
)

type TenantApprovalLogs struct {
	ID        uuid.UUID  `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"column:user_id;not null;type:uuid"`
	Action    *string    `gorm:"column:action;not null"`
	ActionBy  *uuid.UUID `gorm:"column:action_by;not null;type:uuid"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime"`

	// Relations
	User  Users `gorm:"foreignKey:UserID;references:UserID"`
	Actor Users `gorm:"foreignKey:ActionBy;references:UserID"`
}

func (TenantApprovalLogs) TableName() string {
	return "tenant_approval_logs"
}
