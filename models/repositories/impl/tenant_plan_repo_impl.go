package impl

import (
	"backend/models/domains"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantPlanRepoImpl struct{}

func NewTenantPlanRepoImpl() *TenantPlanRepoImpl {
	return &TenantPlanRepoImpl{}
}

func (repo *TenantPlanRepoImpl) Create(db *gorm.DB, tenantPlan domains.TenantPlan) (*domains.TenantPlan, error) {
	if err := db.Create(&tenantPlan).Error; err != nil {
		return nil, err
	}
	return &tenantPlan, nil
}

func (repo *TenantPlanRepoImpl) GetByID(db *gorm.DB, id uuid.UUID) (*domains.TenantPlan, error) {
	var tenantPlan domains.TenantPlan
	if err := db.Preload("Plan").Preload("Tenant").
		Where("id = ?", id).
		First(&tenantPlan).Error; err != nil {
		return nil, err
	}
	return &tenantPlan, nil
}

func (repo *TenantPlanRepoImpl) GetByIDAndTenantID(db *gorm.DB, id, tenantID uuid.UUID) (*domains.TenantPlan, error) {
	var tenantPlan domains.TenantPlan
	if err := db.Preload("Plan").Preload("Tenant").
		Where("id = ? AND tenant_id = ? AND is_paid = false", id, tenantID).
		First(&tenantPlan).Error; err != nil {
		return nil, err
	}
	return &tenantPlan, nil
}

func (repo *TenantPlanRepoImpl) GetByTenantID(db *gorm.DB, tenantID uuid.UUID, pg domains.Pagination) ([]domains.TenantPlan, int, error) {
	var tenantPlans []domains.TenantPlan
	var total int64

	if err := db.Model(&domains.TenantPlan{}).
		Where("tenant_id = ?", tenantID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Preload("Plan").
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Offset(pg.Offset()).
		Limit(pg.Limit).
		Find(&tenantPlans).Error; err != nil {
		return nil, 0, err
	}

	return tenantPlans, int(total), nil
}

func (repo *TenantPlanRepoImpl) GetByStripeSessionID(db *gorm.DB, sessionID string) (*domains.TenantPlan, error) {
	var tenantPlan domains.TenantPlan
	if err := db.Preload("Plan").Preload("Tenant").
		Where("stripe_session_id = ?", sessionID).
		First(&tenantPlan).Error; err != nil {
		return nil, err
	}
	return &tenantPlan, nil
}

// GetLastSequenceToday pakai FOR UPDATE untuk hindari race condition
func (repo *TenantPlanRepoImpl) GetLastSequenceToday(db *gorm.DB) (int, error) {
	var count int64
	today := time.Now().Format("20060102")
	if err := db.Raw(`
		SELECT COUNT(*) FROM (
			SELECT id FROM public.tenant_plan
			WHERE invoice_number LIKE ?
			FOR UPDATE
		) sub`, fmt.Sprintf("INV-AI%s%%", today)).
		Scan(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (repo *TenantPlanRepoImpl) UpdateStripeSession(db *gorm.DB, tenantPlan domains.TenantPlan) error {
	updates := map[string]interface{}{}
	if tenantPlan.StripeSessionID != nil {
		updates["stripe_session_id"] = tenantPlan.StripeSessionID
	}
	if tenantPlan.StripeSessionURL != nil {
		updates["stripe_session_url"] = tenantPlan.StripeSessionURL
	}
	if tenantPlan.StripePaymentStatus != nil {
		updates["stripe_payment_status"] = tenantPlan.StripePaymentStatus
	}
	if tenantPlan.StripeSubscriptionInvoiceID != nil {
		updates["stripe_subscription_invoice_id"] = tenantPlan.StripeSubscriptionInvoiceID
	}
	if len(updates) > 0 {
		if err := db.Model(&domains.TenantPlan{}).
			Where("id = ?", tenantPlan.ID).
			Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func (repo *TenantPlanRepoImpl) UpdatePaymentStatus(db *gorm.DB, tenantPlan domains.TenantPlan) error {
	updates := map[string]interface{}{
		"is_paid": tenantPlan.IsPaid,
	}
	if tenantPlan.PlanStatus != "" {
		updates["plan_status"] = tenantPlan.PlanStatus
	}
	if tenantPlan.StripePaymentStatus != nil {
		updates["stripe_payment_status"] = tenantPlan.StripePaymentStatus
	}
	if tenantPlan.StripePaymentMessage != nil {
		updates["stripe_payment_message"] = tenantPlan.StripePaymentMessage
	}
	if tenantPlan.PaidAt != nil {
		updates["paid_at"] = tenantPlan.PaidAt
	}
	if tenantPlan.StartDate != nil {
		updates["start_date"] = tenantPlan.StartDate
	}
	if tenantPlan.ExpiredDate != nil {
		updates["expired_date"] = tenantPlan.ExpiredDate
	}
	if len(updates) > 0 {
		if err := db.Model(&domains.TenantPlan{}).
			Where("id = ?", tenantPlan.ID).
			Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func (repo *TenantPlanRepoImpl) GetByStripeInvoiceID(db *gorm.DB, invoiceID string) (*domains.TenantPlan, error) {
	var tenantPlan domains.TenantPlan
	if err := db.Preload("Plan").Preload("Tenant").
		Where("stripe_subscription_invoice_id = ?", invoiceID).
		First(&tenantPlan).Error; err != nil {
		return nil, err
	}
	return &tenantPlan, nil
}
