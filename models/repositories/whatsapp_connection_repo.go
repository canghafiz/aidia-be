package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WhatsAppConnectionRepo interface {
	FindByPhoneNumberID(db *gorm.DB, phoneNumberID string) (*domains.WhatsAppConnection, error)
	FindByUserID(db *gorm.DB, userID uuid.UUID) (*domains.WhatsAppConnection, error)
	Upsert(db *gorm.DB, conn domains.WhatsAppConnection) error
	DeleteByUserID(db *gorm.DB, userID uuid.UUID) error
}
