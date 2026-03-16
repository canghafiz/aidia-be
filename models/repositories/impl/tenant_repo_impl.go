package impl

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantRepoImpl struct{}

func NewTenantRepoImpl() *TenantRepoImpl {
	return &TenantRepoImpl{}
}

func (repo *TenantRepoImpl) GetByUserID(db *gorm.DB, userID uuid.UUID) (*domains.Tenant, error) {
	var tenant domains.Tenant
	if err := db.Preload("User").
		Preload("BusinessProfile").
		Where("user_id = ?", userID).
		First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (repo *TenantRepoImpl) GetByTenantID(db *gorm.DB, tenantID uuid.UUID) (*domains.Tenant, error) {
	var tenant domains.Tenant
	if err := db.Preload("User").
		Preload("BusinessProfile").
		Where("tenant_id = ?", tenantID).
		First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}
