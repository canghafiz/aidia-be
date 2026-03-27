package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GuestMessageRepo interface {
	Create(db *gorm.DB, msg domains.GuestMessage) error
	FindByGuestID(db *gorm.DB, guestID uuid.UUID, limit int) ([]domains.GuestMessage, error)
	GetLatestMessages(db *gorm.DB, guestID uuid.UUID, limit int) ([]domains.GuestMessage, error)
}
