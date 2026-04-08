package impl

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WhatsAppConnectionRepoImpl struct{}

func NewWhatsAppConnectionRepoImpl() *WhatsAppConnectionRepoImpl {
	return &WhatsAppConnectionRepoImpl{}
}

func (repo *WhatsAppConnectionRepoImpl) FindByPhoneNumberID(db *gorm.DB, phoneNumberID string) (*domains.WhatsAppConnection, error) {
	var conn domains.WhatsAppConnection
	if err := db.Where("phone_number_id = ?", phoneNumberID).First(&conn).Error; err != nil {
		return nil, err
	}
	return &conn, nil
}

func (repo *WhatsAppConnectionRepoImpl) FindByUserID(db *gorm.DB, userID uuid.UUID) (*domains.WhatsAppConnection, error) {
	var conn domains.WhatsAppConnection
	if err := db.Where("user_id = ?", userID).First(&conn).Error; err != nil {
		return nil, err
	}
	return &conn, nil
}

// Upsert menyimpan atau memperbarui koneksi berdasarkan user_id
func (repo *WhatsAppConnectionRepoImpl) Upsert(db *gorm.DB, conn domains.WhatsAppConnection) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"phone_number_id",
			"waba_id",
			"access_token",
			"phone_number",
			"display_name",
			"connected_at",
			"updated_at",
		}),
	}).Create(&conn).Error
}

func (repo *WhatsAppConnectionRepoImpl) DeleteByUserID(db *gorm.DB, userID uuid.UUID) error {
	return db.Where("user_id = ?", userID).Delete(&domains.WhatsAppConnection{}).Error
}
