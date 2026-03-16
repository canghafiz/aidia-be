package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantPlanRepo interface {
	Create(db *gorm.DB, tenantPlan domains.TenantPlan) (*domains.TenantPlan, error)
	GetByID(db *gorm.DB, id uuid.UUID) (*domains.TenantPlan, error)
	GetByIDAndTenantID(db *gorm.DB, id, tenantID uuid.UUID) (*domains.TenantPlan, error)
	GetByTenantID(db *gorm.DB, tenantID uuid.UUID, pg domains.Pagination) ([]domains.TenantPlan, int, error)
	GetByStripeSessionID(db *gorm.DB, sessionID string) (*domains.TenantPlan, error)
	GetLastSequenceToday(db *gorm.DB) (int, error)
	UpdateStripeSession(db *gorm.DB, tenantPlan domains.TenantPlan) error
	UpdatePaymentStatus(db *gorm.DB, tenantPlan domains.TenantPlan) error
	GetByStripeInvoiceID(db *gorm.DB, invoiceID string) (*domains.TenantPlan, error)
}
