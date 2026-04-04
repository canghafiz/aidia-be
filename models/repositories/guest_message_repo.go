package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GuestMessageRepo interface {
	Create(db *gorm.DB, schema string, msg domains.GuestMessage) error
	FindByGuestID(db *gorm.DB, schema string, guestID uuid.UUID, limit int) ([]domains.GuestMessage, error)
	FindByGuestIDCursor(db *gorm.DB, schema string, guestID uuid.UUID, beforeID *uuid.UUID, limit int) ([]domains.GuestMessage, error)
	GetLatestMessages(db *gorm.DB, schema string, guestID uuid.UUID, limit int) ([]domains.GuestMessage, error)
}
