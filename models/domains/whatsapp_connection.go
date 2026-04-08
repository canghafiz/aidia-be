package domains

import (
	"time"

	"github.com/google/uuid"
)

type WhatsAppConnection struct {
	ID            uuid.UUID `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID        uuid.UUID `gorm:"column:user_id;not null"`
	TenantSchema  string    `gorm:"column:tenant_schema;not null"`
	PhoneNumberID string    `gorm:"column:phone_number_id;not null;uniqueIndex"`
	WabaID        string    `gorm:"column:waba_id;not null"`
	AccessToken   string    `gorm:"column:access_token;not null"`
	PhoneNumber   string    `gorm:"column:phone_number"`
	DisplayName   string    `gorm:"column:display_name"`
	ConnectedAt   time.Time `gorm:"column:connected_at;autoCreateTime"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (WhatsAppConnection) TableName() string {
	return "public.whatsapp_connections"
}
