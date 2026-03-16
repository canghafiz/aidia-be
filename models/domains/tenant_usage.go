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

// UsedTokens mengembalikan jumlah token yang sudah digunakan.
// Untuk free plan: TotalTokens (1jt) - sisa token yang tersimpan di DB
// Untuk paid plan (TotalTokens = -1): ambil dari field terpisah atau hitung dari cost
// Karena kita simpan total_tokens sebagai sisa token untuk free,
// dan -1 untuk paid, maka used = 1jt - TotalTokens untuk free.
// Untuk paid, used tokens perlu disimpan di field lain — untuk sekarang return 0
// sampai ada field used_tokens di DB.
func (u TenantUsage) UsedTokens() int64 {
	if u.TotalTokens == -1 {
		// unlimited — belum ada field used_tokens, return 0 dulu
		return 0
	}
	// free: awalnya 1jt, sisa = TotalTokens, used = 1jt - sisa
	initial := int64(1_000_000)
	used := initial - u.TotalTokens
	if used < 0 {
		return 0
	}
	return used
}
