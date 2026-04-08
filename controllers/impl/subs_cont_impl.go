package impl

import (
	"backend/exceptions"
	"backend/helpers"
	"backend/models/services"

	"github.com/gin-gonic/gin"
)

type SubsContImpl struct {
	SubsServ services.SubsServ
}

func NewSubsContImpl(subsServ services.SubsServ) *SubsContImpl {
	return &SubsContImpl{SubsServ: subsServ}
}

// GetCurrentSubs @Summary      Get Current Subscription
// @Description  Ambil status subscription aktif tenant yang sedang login.
// @Description  Mengembalikan info plan aktif (single/multiple), token usage, dan pesan status.
// @Description  Jika tidak ada plan aktif, kembalikan info free plan usage.
// @Tags         Subscription
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  helpers.ApiResponse{data=subs.Response}
// @Failure      401  {object}  helpers.ApiResponse
// @Failure      500  {object}  helpers.ApiResponse
// @Router       /subs/current [get]
func (cont *SubsContImpl) GetCurrentSubs(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	result, err := cont.SubsServ.GetCurrentSubs(accessToken)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    result,
	}

	errResponse := helpers.WriteToResponseBody(ctx, response.Code, response)
	if errResponse != nil {
		exceptions.ErrorHandler(ctx, errResponse)
		return
	}
}

// GetTokenUsage godoc
// @Summary      Get AI Token Usage
// @Description  Returns the current AI token usage for the authenticated tenant.
// @Description
// @Description  **Free Plan:**
// @Description  - `plan_type` = "free"
// @Description  - `is_unlimited` = false
// @Description  - `token_limit` = 1,000,000
// @Description  - `tokens_used` = total tokens consumed so far
// @Description  - `tokens_remaining` = tokens left before the bot is blocked
// @Description  - `percentage_used` = percentage of quota used (0–100)
// @Description  - `message` shows a warning when usage ≥ 80%, and an upgrade prompt when limit is reached
// @Description
// @Description  **Paid Plan:**
// @Description  - `plan_type` = "paid"
// @Description  - `is_unlimited` = true
// @Description  - All numeric fields return -1 (unlimited)
// @Description
// @Description  **Error Responses:**
// @Description  - 401 — Token missing, expired, or invalid
// @Description  - 403 — Authenticated user is not a Client
// @Description  - 404 — Tenant record not found
// @Description  - 500 — Failed to retrieve subscription or usage data
// @Tags         Subscription
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  helpers.ApiResponse{data=subs.TokenUsageResponse}  "Token usage data"
// @Failure      401  {object}  helpers.ApiResponse                                 "Unauthorized — invalid or missing token"
// @Failure      403  {object}  helpers.ApiResponse                                 "Forbidden — user is not a Client"
// @Failure      404  {object}  helpers.ApiResponse                                 "Tenant not found"
// @Failure      500  {object}  helpers.ApiResponse                                 "Internal server error"
// @Router       /api/v1/subs/token-usage [get]
func (cont *SubsContImpl) GetTokenUsage(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	result, err := cont.SubsServ.GetTokenUsage(accessToken)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    result,
	}

	errResponse := helpers.WriteToResponseBody(ctx, response.Code, response)
	if errResponse != nil {
		exceptions.ErrorHandler(ctx, errResponse)
		return
	}
}
