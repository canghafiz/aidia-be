package domains

import (
	"time"

	"github.com/google/uuid"
)

type TenantUsage struct {
	ID           uuid.UUID  `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     uuid.UUID  `gorm:"column:tenant_id;not null"`
	TenantPlanID *uuid.UUID `gorm:"column:tenant_plan_id;default:null"`
	Period       time.Time  `gorm:"column:period;not null"`
	TotalTokens  int64      `gorm:"column:total_tokens;default:0"`
	TotalCost    float64    `gorm:"column:total_cost;default:0"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime"`

	Tenant     *Tenant     `gorm:"foreignKey:TenantID;references:TenantID"`
	TenantPlan *TenantPlan `gorm:"foreignKey:TenantPlanID;references:ID"`
}

func (TenantUsage) TableName() string {
	return "public.tenant_usage"
}

// UsedTokens returns the number of tokens consumed so far.
// Free plan: TotalTokens holds the remaining balance (started at 1M), so used = 1M - TotalTokens.
// Paid plan (TotalTokens = -1): tokens are unlimited; return 0 until a dedicated used_tokens field exists.
func (u TenantUsage) UsedTokens() int64 {
	if u.TotalTokens == -1 {
		// unlimited — no used_tokens field yet, return 0 for now
		return 0
	}
	// free: initial = 1M, remaining = TotalTokens, used = 1M - remaining
	initial := int64(1_000_000)
	used := initial - u.TotalTokens
	if used < 0 {
		return 0
	}
	return used
}
