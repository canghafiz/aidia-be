package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GuestRepo interface {
	Create(db *gorm.DB, schema string, guest domains.Guest) error
	FindByID(db *gorm.DB, schema string, id uuid.UUID) (*domains.Guest, error)
	FindByPlatformChatID(db *gorm.DB, schema, chatID string) (*domains.Guest, error)
	FindAllByTenantID(db *gorm.DB, schema string, tenantID uuid.UUID, pagination domains.Pagination) ([]domains.Guest, int64, error)
	MarkAsRead(db *gorm.DB, schema string, guestID uuid.UUID) error
	Update(db *gorm.DB, schema string, guest domains.Guest) error
}
