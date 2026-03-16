package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantUsageRepo interface {
	// GetFreeUsageByTenantID ambil row free usage (tenant_plan_id = NULL)
	GetFreeUsageByTenantID(db *gorm.DB, tenantID uuid.UUID) (*domains.TenantUsage, error)

	// GetActiveUsageByTenantID ambil semua usage row yang tenant_plan-nya masih Active
	GetActiveUsageByTenantID(db *gorm.DB, tenantID uuid.UUID) ([]domains.TenantUsage, error)

	// GetByTenantPlanID ambil usage row berdasarkan tenant_plan_id
	GetByTenantPlanID(db *gorm.DB, tenantPlanID uuid.UUID) (*domains.TenantUsage, error)

	// UpdateUsage update total_tokens dan total_cost
	UpdateUsage(db *gorm.DB, tenantUsage domains.TenantUsage) error
}
