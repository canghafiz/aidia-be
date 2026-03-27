package impl

import (
	"backend/models/domains"
	"backend/models/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GuestRepoImpl struct{}

func NewGuestRepoImpl() *GuestRepoImpl {
	return &GuestRepoImpl{}
}

func (repo *GuestRepoImpl) Create(db *gorm.DB, guest domains.Guest) error {
	return db.Table(guest.TableName()).Create(&guest).Error
}

func (repo *GuestRepoImpl) FindByID(db *gorm.DB, id uuid.UUID) (*domains.Guest, error) {
	var guest domains.Guest
	err := db.Table(guest.TableName()).Where("id = ?", id).First(&guest).Error
	if err != nil {
		return nil, err
	}
	return &guest, nil
}

func (repo *GuestRepoImpl) FindByTelegramChatID(db *gorm.DB, schema, chatID string) (*domains.Guest, error) {
	var guest domains.Guest
	err := db.Table(schema + "." + guest.TableName()).
		Where("telegram_chat_id = ?", chatID).
		First(&guest).Error
	if err != nil {
		return nil, err
	}
	return &guest, nil
}

func (repo *GuestRepoImpl) Update(db *gorm.DB, guest domains.Guest) error {
	return db.Table(guest.TableName()).Save(&guest).Error
}

var _ repositories.GuestRepo = (*GuestRepoImpl)(nil)
