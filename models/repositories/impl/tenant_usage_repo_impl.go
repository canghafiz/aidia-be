package impl

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantUsageRepoImpl struct{}

func NewTenantUsageRepoImpl() *TenantUsageRepoImpl {
	return &TenantUsageRepoImpl{}
}

func (repo *TenantUsageRepoImpl) GetFreeUsageByTenantID(db *gorm.DB, tenantID uuid.UUID) (*domains.TenantUsage, error) {
	var tenantUsage domains.TenantUsage
	if err := db.Where("tenant_id = ? AND tenant_plan_id IS NULL", tenantID).
		First(&tenantUsage).Error; err != nil {
		return nil, err
	}
	return &tenantUsage, nil
}

func (repo *TenantUsageRepoImpl) GetActiveUsageByTenantID(db *gorm.DB, tenantID uuid.UUID) ([]domains.TenantUsage, error) {
	var usages []domains.TenantUsage
	if err := db.
		Joins("JOIN public.tenant_plan tp ON tp.id = public.tenant_usage.tenant_plan_id").
		Where("public.tenant_usage.tenant_id = ? AND tp.plan_status = 'Active' AND tp.expired_date >= NOW()::DATE", tenantID).
		Preload("TenantPlan").
		Preload("TenantPlan.Plan").
		Find(&usages).Error; err != nil {
		return nil, err
	}
	return usages, nil
}

func (repo *TenantUsageRepoImpl) GetByTenantPlanID(db *gorm.DB, tenantPlanID uuid.UUID) (*domains.TenantUsage, error) {
	var tenantUsage domains.TenantUsage
	if err := db.Where("tenant_plan_id = ?", tenantPlanID).
		Preload("TenantPlan").
		Preload("TenantPlan.Plan").
		First(&tenantUsage).Error; err != nil {
		return nil, err
	}
	return &tenantUsage, nil
}

func (repo *TenantUsageRepoImpl) UpdateUsage(db *gorm.DB, tenantUsage domains.TenantUsage) error {
	updates := map[string]interface{}{}
	updates["total_tokens"] = tenantUsage.TotalTokens
	updates["total_cost"] = tenantUsage.TotalCost

	return db.Model(&domains.TenantUsage{}).
		Where("id = ?", tenantUsage.ID).
		Updates(updates).Error
}
