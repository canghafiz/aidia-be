// models/domains/tenant_plan.go
package domains

import (
	"time"

	"github.com/google/uuid"
)

type TenantPlan struct {
	ID                          uuid.UUID  `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID                    uuid.UUID  `gorm:"column:tenant_id;not null;type:uuid"`
	PlanID                      uuid.UUID  `gorm:"column:plan_id;not null;type:uuid"`
	InvoiceNumber               string     `gorm:"column:invoice_number;not null;uniqueIndex:uq_tenant_plan_invoice_number"`
	Duration                    int        `gorm:"column:duration;not null"`
	IsMonth                     bool       `gorm:"column:is_month;not null"`
	Price                       float64    `gorm:"column:price;not null"`
	PaymentDueDate              *time.Time `gorm:"column:payment_due_date"`
	PaidAt                      *time.Time `gorm:"column:paid_at"`
	StartDate                   *time.Time `gorm:"column:start_date"`
	ExpiredDate                 *time.Time `gorm:"column:expired_date"`
	PlanStatus                  string     `gorm:"column:plan_status;not null;default:Inactive"`
	StripeSessionID             *string    `gorm:"column:stripe_session_id"`
	StripeSessionURL            *string    `gorm:"column:stripe_session_url"`
	StripePaymentStatus         *string    `gorm:"column:stripe_payment_status"`
	StripePaymentMessage        *string    `gorm:"column:stripe_payment_message"`
	StripeSubscriptionInvoiceID *string    `gorm:"column:stripe_subscription_invoice_id"`
	IsPaid                      bool       `gorm:"column:is_paid;not null;default:false"`
	IsActive                    bool       `gorm:"column:is_active;not null;default:true"`
	CreatedAt                   time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt                   time.Time  `gorm:"column:updated_at;autoUpdateTime"`

	// Relations
	Tenant Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
	Plan   Plan   `gorm:"foreignKey:PlanID;references:ID"`
}

func (TenantPlan) TableName() string {
	return "public.tenant_plan"
}
