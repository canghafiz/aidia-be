package impl

import (
	"backend/models/domains"
	"backend/models/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GuestMessageRepoImpl struct{}

func NewGuestMessageRepoImpl() *GuestMessageRepoImpl {
	return &GuestMessageRepoImpl{}
}

func (repo *GuestMessageRepoImpl) Create(db *gorm.DB, schema string, msg domains.GuestMessage) error {
	return db.Table(schema + "." + msg.TableName()).Create(&msg).Error
}

func (repo *GuestMessageRepoImpl) FindByGuestID(db *gorm.DB, schema string, guestID uuid.UUID, limit int) ([]domains.GuestMessage, error) {
	var messages []domains.GuestMessage
	err := db.Table(schema + ".guest_message").
		Where("guest_id = ?", guestID).
		Order("created_at ASC").
		Limit(limit).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (repo *GuestMessageRepoImpl) GetLatestMessages(db *gorm.DB, schema string, guestID uuid.UUID, limit int) ([]domains.GuestMessage, error) {
	var messages []domains.GuestMessage
	err := db.Table(schema + ".guest_message").
		Where("guest_id = ?", guestID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

var _ repositories.GuestMessageRepo = (*GuestMessageRepoImpl)(nil)
