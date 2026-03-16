package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantRepo interface {
	GetByUserID(db *gorm.DB, userID uuid.UUID) (*domains.Tenant, error)
	GetByTenantID(db *gorm.DB, tenantID uuid.UUID) (*domains.Tenant, error)
}
