package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GuestRepo interface {
	Create(db *gorm.DB, schema string, guest domains.Guest) error
	FindByID(db *gorm.DB, schema string, id uuid.UUID) (*domains.Guest, error)
	FindByTelegramChatID(db *gorm.DB, schema, chatID string) (*domains.Guest, error)
	Update(db *gorm.DB, schema string, guest domains.Guest) error
}
