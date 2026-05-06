package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantUsageRepo interface {
	// GetFreeUsageByTenantID fetches the free usage row (tenant_plan_id = NULL)
	GetFreeUsageByTenantID(db *gorm.DB, tenantID uuid.UUID) (*domains.TenantUsage, error)

	// GetActiveUsageByTenantID fetches all usage rows whose tenant_plan is still Active
	GetActiveUsageByTenantID(db *gorm.DB, tenantID uuid.UUID) ([]domains.TenantUsage, error)

	// GetByTenantPlanID fetches a usage row by tenant_plan_id
	GetByTenantPlanID(db *gorm.DB, tenantPlanID uuid.UUID) (*domains.TenantUsage, error)

	// UpdateUsage updates total_tokens and total_cost
	UpdateUsage(db *gorm.DB, tenantUsage domains.TenantUsage) error
}
