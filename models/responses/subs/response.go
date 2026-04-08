package subs

import (
	"backend/models/domains"
	"fmt"
	"time"
)

// ============================================================
// TOKEN USAGE
// ============================================================

type TokenUsage struct {
	TotalTokens int64   `json:"total_tokens"` // -1 = unlimited
	UsedTokens  int64   `json:"used_tokens"`
	TotalCost   float64 `json:"total_cost"`
}

// TokenUsageResponse is the dedicated response for the token-usage endpoint.
type TokenUsageResponse struct {
	PlanType         string  `json:"plan_type"`          // "free" | "paid"
	IsUnlimited      bool    `json:"is_unlimited"`       // true if paid plan active
	TokenLimit       int64   `json:"token_limit"`        // 1_000_000 for free, -1 for unlimited
	TokensUsed       int64   `json:"tokens_used"`        // total tokens consumed
	TokensRemaining  int64   `json:"tokens_remaining"`   // -1 if unlimited
	PercentageUsed   float64 `json:"percentage_used"`    // 0-100, -1 if unlimited
	Message          string  `json:"message"`            // human-readable summary
}

// ============================================================
// PLAN INFO
// ============================================================

type ActivePlanInfo struct {
	TenantPlanID string     `json:"tenant_plan_id"`
	PlanName     string     `json:"plan_name"`
	Duration     string     `json:"duration"`
	ValidFrom    string     `json:"valid_from"`
	ValidUntil   string     `json:"valid_until"`
	TokenUsage   TokenUsage `json:"token_usage"`
}

// ============================================================
// SUBS RESPONSE
// ============================================================

type Response struct {
	IsFree            bool             `json:"is_free"`
	Message           string           `json:"message"`
	ActivePlans       []ActivePlanInfo `json:"active_plans"`
	CurrentTokenUsage TokenUsage       `json:"current_token_usage"`
}

// ============================================================
// HELPER
// ============================================================

func formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("02 Jan 2006")
}

func formatDuration(duration int, isMonth bool) string {
	if isMonth {
		if duration == 1 {
			return "1 Month"
		}
		return fmt.Sprintf("%d Months", duration)
	}
	if duration == 1 {
		return "1 Year"
	}
	return fmt.Sprintf("%d Years", duration)
}

func buildMessage(plans []ActivePlanInfo) string {
	if len(plans) == 0 {
		return "You are currently on the free plan"
	}
	if len(plans) == 1 {
		p := plans[0]
		return fmt.Sprintf("Your current plan is %s - %s, valid from %s to %s",
			p.PlanName, p.Duration, p.ValidFrom, p.ValidUntil)
	}
	return fmt.Sprintf("You have %d active plans", len(plans))
}

// ============================================================
// MAPPER
// ============================================================

func ToSubsResponse(
	activeUsages []domains.TenantUsage,
	freeUsage *domains.TenantUsage,
) Response {
	isFree := len(activeUsages) == 0

	var activePlans []ActivePlanInfo
	var totalUsedTokens int64
	var totalCost float64

	for _, u := range activeUsages {
		if u.TenantPlan == nil {
			continue
		}

		tp := u.TenantPlan
		plan := tp.Plan

		// used_tokens = selisih dari total_tokens awal (unlimited = -1, jadi hitung dari cost saja)
		// Di sini kita simpan used_tokens sebagai nilai positif dari penggunaan
		usedTokens := u.UsedTokens()
		totalUsedTokens += usedTokens
		totalCost += u.TotalCost

		activePlans = append(activePlans, ActivePlanInfo{
			TenantPlanID: tp.ID.String(),
			PlanName:     plan.Name,
			Duration:     formatDuration(tp.Duration, tp.IsMonth),
			ValidFrom:    formatDate(tp.StartDate),
			ValidUntil:   formatDate(tp.ExpiredDate),
			TokenUsage: TokenUsage{
				TotalTokens: -1,
				UsedTokens:  usedTokens,
				TotalCost:   u.TotalCost,
			},
		})
	}

	// Current token usage
	var currentTokenUsage TokenUsage
	if isFree && freeUsage != nil {
		usedTokens := freeUsage.UsedTokens()
		currentTokenUsage = TokenUsage{
			TotalTokens: freeUsage.TotalTokens,
			UsedTokens:  usedTokens,
			TotalCost:   freeUsage.TotalCost,
		}
	} else {
		currentTokenUsage = TokenUsage{
			TotalTokens: -1,
			UsedTokens:  totalUsedTokens,
			TotalCost:   totalCost,
		}
	}

	return Response{
		IsFree:            isFree,
		Message:           buildMessage(activePlans),
		ActivePlans:       activePlans,
		CurrentTokenUsage: currentTokenUsage,
	}
}
