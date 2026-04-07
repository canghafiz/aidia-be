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

// FindByGuestIDCursor loads messages older than beforeID (cursor-based lazy load).
// If beforeID is nil, returns the most recent messages. Results are DESC (newest first).
// If platform is non-empty, only messages from that platform are returned.
func (repo *GuestMessageRepoImpl) FindByGuestIDCursor(db *gorm.DB, schema string, guestID uuid.UUID, platform string, beforeID *uuid.UUID, limit int) ([]domains.GuestMessage, error) {
	var messages []domains.GuestMessage
	q := db.Table(schema+".guest_message").Where("guest_id = ?", guestID)
	if platform != "" {
		q = q.Where("platform = ?", platform)
	}
	if beforeID != nil {
		// Fetch messages with created_at before the cursor message
		q = q.Where("created_at < (SELECT created_at FROM "+schema+".guest_message WHERE id = ?)", beforeID)
	}
	err := q.Order("created_at DESC").Limit(limit).Find(&messages).Error
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
