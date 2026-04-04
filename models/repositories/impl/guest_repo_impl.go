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

func (repo *GuestRepoImpl) Create(db *gorm.DB, schema string, guest domains.Guest) error {
	return db.Table(schema + "." + guest.TableName()).Create(&guest).Error
}

func (repo *GuestRepoImpl) FindByID(db *gorm.DB, schema string, id uuid.UUID) (*domains.Guest, error) {
	var guest domains.Guest
	err := db.Table(schema + "." + guest.TableName()).Where("id = ?", id).First(&guest).Error
	if err != nil {
		return nil, err
	}
	return &guest, nil
}

func (repo *GuestRepoImpl) FindByPlatformChatID(db *gorm.DB, schema, chatID string) (*domains.Guest, error) {
	var guest domains.Guest
	err := db.Table(schema + "." + guest.TableName()).
		Where("platform_chat_id = ?", chatID).
		First(&guest).Error
	if err != nil {
		return nil, err
	}
	return &guest, nil
}

func (repo *GuestRepoImpl) FindAllByTenantID(db *gorm.DB, schema string, tenantID uuid.UUID, pagination domains.Pagination) ([]domains.Guest, int64, error) {
	var guests []domains.Guest
	var total int64

	table := schema + ".guest"
	db.Table(table).Where("tenant_id = ?", tenantID).Count(&total)

	err := db.Table(table).
		Where("tenant_id = ?", tenantID).
		Order("last_message_at DESC NULLS LAST").
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Find(&guests).Error
	if err != nil {
		return nil, 0, err
	}
	return guests, total, nil
}

func (repo *GuestRepoImpl) MarkAsRead(db *gorm.DB, schema string, guestID uuid.UUID) error {
	return db.Table(schema+".guest").
		Where("id = ?", guestID).
		Update("is_read", true).Error
}

func (repo *GuestRepoImpl) Update(db *gorm.DB, schema string, guest domains.Guest) error {
	return db.Table(schema + "." + guest.TableName()).Save(&guest).Error
}

var _ repositories.GuestRepo = (*GuestRepoImpl)(nil)
