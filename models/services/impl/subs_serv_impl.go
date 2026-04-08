package impl

import (
	"backend/helpers"
	"backend/models/repositories"
	subsRes "backend/models/responses/subs"
	"fmt"
	"log"
	"math"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubsServImpl struct {
	Db              *gorm.DB
	JwtKey          string
	TenantRepo      repositories.TenantRepo
	TenantUsageRepo repositories.TenantUsageRepo
}

func NewSubsServImpl(
	db *gorm.DB,
	jwtKey string,
	tenantRepo repositories.TenantRepo,
	tenantUsageRepo repositories.TenantUsageRepo,
) *SubsServImpl {
	return &SubsServImpl{
		Db:              db,
		JwtKey:          jwtKey,
		TenantRepo:      tenantRepo,
		TenantUsageRepo: tenantUsageRepo,
	}
}

// GetTokenUsage returns the AI token usage for the authenticated tenant.
// Free plan: shows tokens used, remaining, and percentage.
// Paid plan: shows unlimited status.
func (serv *SubsServImpl) GetTokenUsage(accessToken string) (*subsRes.TokenUsageResponse, error) {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"Client"})
	if err != nil || !ok {
		return nil, fmt.Errorf("user role not authorized")
	}

	userIDStr, err := helpers.GetUserIdFromToken(accessToken, serv.JwtKey)
	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}
	userID, _ := uuid.Parse(*userIDStr)

	tenant, err := serv.TenantRepo.GetByUserID(serv.Db, userID)
	if err != nil {
		return nil, fmt.Errorf("tenant not found")
	}

	// Check if there is an active paid subscription
	activeUsages, err := serv.TenantUsageRepo.GetActiveUsageByTenantID(serv.Db, tenant.TenantID)
	if err != nil {
		log.Printf("[SubsServ] GetActiveUsageByTenantID error: %v", err)
		return nil, fmt.Errorf("failed to retrieve subscription data")
	}

	hasPaidPlan := len(activeUsages) > 0

	if hasPaidPlan {
		return &subsRes.TokenUsageResponse{
			PlanType:        "paid",
			IsUnlimited:     true,
			TokenLimit:      -1,
			TokensUsed:      -1,
			TokensRemaining: -1,
			PercentageUsed:  -1,
			Message:         "You are on a paid plan. AI usage is unlimited.",
		}, nil
	}

	// Free plan — fetch usage record
	freeUsage, err := serv.TenantUsageRepo.GetFreeUsageByTenantID(serv.Db, tenant.TenantID)
	if err != nil || freeUsage == nil {
		// No usage record yet → full quota available
		return &subsRes.TokenUsageResponse{
			PlanType:        "free",
			IsUnlimited:     false,
			TokenLimit:      1_000_000,
			TokensUsed:      0,
			TokensRemaining: 1_000_000,
			PercentageUsed:  0,
			Message:         "You are on the free plan. 1,000,000 tokens available.",
		}, nil
	}

	const limit int64 = 1_000_000
	tokensUsed := limit - freeUsage.TotalTokens
	if tokensUsed < 0 {
		tokensUsed = 0
	}
	tokensRemaining := freeUsage.TotalTokens
	if tokensRemaining < 0 {
		tokensRemaining = 0
	}
	pct := math.Round(float64(tokensUsed)/float64(limit)*10000) / 100

	var message string
	switch {
	case tokensRemaining == 0:
		message = "Your free AI token limit has been reached. Please upgrade to a paid plan to continue using the AI assistant."
	case pct >= 80:
		message = fmt.Sprintf("Warning: You have used %.2f%% of your free tokens. Consider upgrading soon.", pct)
	default:
		message = fmt.Sprintf("You have used %d of %d free tokens (%.2f%%).", tokensUsed, limit, pct)
	}

	return &subsRes.TokenUsageResponse{
		PlanType:        "free",
		IsUnlimited:     false,
		TokenLimit:      limit,
		TokensUsed:      tokensUsed,
		TokensRemaining: tokensRemaining,
		PercentageUsed:  pct,
		Message:         message,
	}, nil
}

func (serv *SubsServImpl) GetCurrentSubs(accessToken string) (*subsRes.Response, error) {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"Client"})
	if err != nil || !ok {
		return nil, err
	}

	userIDStr, err := helpers.GetUserIdFromToken(accessToken, serv.JwtKey)
	if err != nil {
		return nil, err
	}
	userID, _ := uuid.Parse(*userIDStr)

	tenant, err := serv.TenantRepo.GetByUserID(serv.Db, userID)
	if err != nil {
		return nil, fmt.Errorf("tenant not found")
	}

	// Ambil semua usage dari plan yang masih aktif
	activeUsages, err := serv.TenantUsageRepo.GetActiveUsageByTenantID(serv.Db, tenant.TenantID)
	if err != nil {
		log.Printf("[SubsServ] GetActiveUsageByTenantID error: %v", err)
		return nil, fmt.Errorf("failed to get active subscriptions")
	}

	// Ambil free usage (tenant_plan_id = NULL)
	freeUsage, err := serv.TenantUsageRepo.GetFreeUsageByTenantID(serv.Db, tenant.TenantID)
	if err != nil {
		log.Printf("[SubsServ] GetFreeUsageByTenantID error: %v", err)
		// Tidak return error — free usage mungkin belum ada
		freeUsage = nil
	}

	response := subsRes.ToSubsResponse(activeUsages, freeUsage)
	return &response, nil
}
