package domains

import (
	"time"

	"github.com/google/uuid"
)

type GuestMessage struct {
	ID                uuid.UUID `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	GuestID           uuid.UUID `gorm:"column:guest_id;not null;type:uuid;index:idx_guest_msg_guest"`
	Role              string    `gorm:"column:role;not null"`
	Type              string    `gorm:"column:type"`
	Message           string    `gorm:"column:message"`
	IsHuman           bool      `gorm:"column:is_human;not null;default:false"`
	IsActive          bool      `gorm:"column:is_active;not null;default:true"`
	PlatformMessageID *int      `gorm:"column:platform_message_id;index:idx_guest_msg_platform_id"`
	Platform          string    `gorm:"column:platform;not null;default:'telegram';index:idx_guest_msg_platform"`
	SessionID         string    `gorm:"column:session_id;index:idx_guest_msg_session"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime;index:idx_guest_msg_created"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (GuestMessage) TableName() string {
	return "guest_message"
}
