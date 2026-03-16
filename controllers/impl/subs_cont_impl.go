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
