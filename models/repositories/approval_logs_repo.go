package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ApprovalLogsRepo interface {
	Approve(db *gorm.DB, approve domains.TenantApprovalLogs) error
	GetAll(db *gorm.DB, pagination domains.Pagination) ([]domains.TenantApprovalLogs, int, error)
	GetByID(db *gorm.DB, id uuid.UUID) (*domains.TenantApprovalLogs, error)
	Delete(db *gorm.DB, logId uuid.UUID) error
}
