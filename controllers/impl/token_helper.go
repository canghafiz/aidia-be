package impl

import (
	"backend/models/domains"
	"backend/models/repositories"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const FreeTokenLimit int64 = 1_000_000

// hasActiveSubs returns true if the tenant has at least one active paid subscription.
func hasActiveSubs(db *gorm.DB, tenantUsageRepo repositories.TenantUsageRepo, tenantID uuid.UUID) bool {
	usages, err := tenantUsageRepo.GetActiveUsageByTenantID(db, tenantID)
	return err == nil && len(usages) > 0
}

// freeTokensRemaining returns remaining free tokens for a tenant.
// Returns FreeTokenLimit if no usage record exists yet (brand new tenant).
func freeTokensRemaining(db *gorm.DB, tenantUsageRepo repositories.TenantUsageRepo, tenantID uuid.UUID) int64 {
	usage, err := tenantUsageRepo.GetFreeUsageByTenantID(db, tenantID)
	if err != nil || usage == nil {
		return FreeTokenLimit
	}
	return usage.TotalTokens
}

// deductFreeTokens deducts used tokens from the tenant's free usage record.
// Creates the record with the remaining balance if it doesn't exist yet.
func deductFreeTokens(db *gorm.DB, tenantUsageRepo repositories.TenantUsageRepo, tenantID uuid.UUID, tokens int64) {
	usage, err := tenantUsageRepo.GetFreeUsageByTenantID(db, tenantID)
	if err != nil || usage == nil {
		// First-time usage — create the free usage row
		remaining := FreeTokenLimit - tokens
		if remaining < 0 {
			remaining = 0
		}
		newUsage := domains.TenantUsage{
			TenantID:    tenantID,
			Period:      time.Now(),
			TotalTokens: remaining,
			TotalCost:   0,
		}
		if err := db.Create(&newUsage).Error; err != nil {
			log.Printf("[Token] failed to create free usage record for tenant %s: %v", tenantID, err)
		}
		return
	}

	usage.TotalTokens -= tokens
	if usage.TotalTokens < 0 {
		usage.TotalTokens = 0
	}
	if err := tenantUsageRepo.UpdateUsage(db, *usage); err != nil {
		log.Printf("[Token] failed to update usage for tenant %s: %v", tenantID, err)
	}
}
