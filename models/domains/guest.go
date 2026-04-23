package domains

import (
	"time"

	"github.com/google/uuid"
)

type Guest struct {
	ID                uuid.UUID  `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID          *uuid.UUID `gorm:"column:tenant_id;type:uuid"`
	Identity          string     `gorm:"column:identity;not null;index:idx_guest_identity"`
	Username          string     `gorm:"column:username"`
	Phone             string     `gorm:"column:phone"`
	Name              string     `gorm:"column:name"`
	Sosmed            JSONB      `gorm:"column:sosmed;type:jsonb"`
	AiThreadID        string     `gorm:"column:ai_thread_id"`
	IsTakeOver        bool       `gorm:"column:is_take_over;not null;default:false"`
	IsRead            bool       `gorm:"column:is_read;not null;default:false"`
	IsActive          bool       `gorm:"column:is_active;not null;default:true"`
	Platform          string     `gorm:"column:platform"`
	PlatformChatID    string     `gorm:"column:platform_chat_id;index:idx_guest_platform_chat"`
	LastMessageAt     *time.Time `gorm:"column:last_message_at;index:idx_guest_last_message"`
	ConversationState JSONB      `gorm:"column:conversation_state;type:jsonb"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (Guest) TableName() string {
	return "guest"
}
