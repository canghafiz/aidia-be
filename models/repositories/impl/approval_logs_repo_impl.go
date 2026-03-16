package impl

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ApprovalLogsRepoImpl struct {
}

func NewApprovalLogsRepoImpl() *ApprovalLogsRepoImpl {
	return &ApprovalLogsRepoImpl{}
}

func (repo *ApprovalLogsRepoImpl) Approve(db *gorm.DB, approve domains.TenantApprovalLogs) error {
	updates := map[string]interface{}{}

	if approve.Action != nil && *approve.Action != "" {
		updates["action"] = approve.Action
	}
	if approve.ActionBy != nil && *approve.ActionBy != (uuid.UUID{}) {
		updates["action_by"] = approve.ActionBy
	}

	if len(updates) > 0 {
		if err := db.Model(&domains.TenantApprovalLogs{}).
			Where("id = ?", approve.ID).
			Updates(updates).Error; err != nil {
			return err
		}
	}

	return nil
}

func (repo *ApprovalLogsRepoImpl) GetAll(db *gorm.DB, pagination domains.Pagination) ([]domains.TenantApprovalLogs, int, error) {
	var logs []domains.TenantApprovalLogs
	var total int64

	base := db.Model(&domains.TenantApprovalLogs{})

	// Count total
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Preload
	if err := db.Model(&domains.TenantApprovalLogs{}).
		Preload("User").
		Preload("User.Tenant").
		Preload("User.Tenant.BusinessProfile").
		Preload("Actor").
		Offset(pagination.Offset()).
		Limit(pagination.Limit).
		Order("created_at DESC").
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, int(total), nil
}

func (repo *ApprovalLogsRepoImpl) GetByID(db *gorm.DB, id uuid.UUID) (*domains.TenantApprovalLogs, error) {
	var log domains.TenantApprovalLogs
	if err := db.Where("id = ?", id).First(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

func (repo *ApprovalLogsRepoImpl) Delete(db *gorm.DB, logId uuid.UUID) error {
	if err := db.
		Where("id = ?", logId).
		Delete(&domains.TenantApprovalLogs{}).Error; err != nil {
		return err
	}
	return nil
}
