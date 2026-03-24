package impl

import (
	"backend/exceptions"
	"backend/helpers"
	"backend/models/domains"
	reqOP "backend/models/requests/order_payment"
	"backend/models/services"

	"github.com/gin-gonic/gin"
)

type OrderPaymentContImpl struct {
	OrderPaymentServ services.OrderPaymentServ
}

func NewOrderPaymentContImpl(orderPaymentServ services.OrderPaymentServ) *OrderPaymentContImpl {
	return &OrderPaymentContImpl{OrderPaymentServ: orderPaymentServ}
}

// GetAll @Summary      Get All Order Payments
// @Description  Ambil semua order payment dengan pagination
// @Tags         Order Payment
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path   string  true  "Client ID"
// @Param        page       query  int     false "Page"
// @Param        limit      query  int     false "Limit"
// @Success      200        {object}  helpers.ApiResponse{data=pagination.Response}
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/order-payments [get]
func (cont *OrderPaymentContImpl) GetAll(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	pg := domains.ParsePagination(ctx)

	result, err := cont.OrderPaymentServ.GetAll(accessToken, clientID, pg)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: result}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}

// GetByID @Summary      Get Order Payment By ID
// @Description  Ambil detail order payment berdasarkan ID
// @Tags         Order Payment
// @Produce      json
// @Security     BearerAuth
// @Param        client_id   path  string true "Client ID"
// @Param        payment_id  path  string true "Payment ID"
// @Success      200         {object}  helpers.ApiResponse{data=order_payment.OrderPaymentResponse}
// @Failure      400         {object}  helpers.ApiResponse
// @Failure      401         {object}  helpers.ApiResponse
// @Failure      500         {object}  helpers.ApiResponse
// @Router       /client/{client_id}/order-payments/{payment_id} [get]
func (cont *OrderPaymentContImpl) GetByID(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	paymentID, err := helpers.ParseUUID(ctx, "payment_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, err := cont.OrderPaymentServ.GetByID(accessToken, clientID, paymentID)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: result}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}

// UpdateStatus @Summary      Update Order Payment Status
// @Description  Update status order payment
// @Tags         Order Payment
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id   path  string                                    true "Client ID"
// @Param        payment_id  path  string                                    true "Payment ID"
// @Param        request     body  order_payment.UpdatePaymentStatusRequest  true "Update Payment Status Request"
// @Success      200         {object}  helpers.ApiResponse
// @Failure      400         {object}  helpers.ApiResponse
// @Failure      401         {object}  helpers.ApiResponse
// @Failure      500         {object}  helpers.ApiResponse
// @Router       /client/{client_id}/order-payments/{payment_id}/status [patch]
func (cont *OrderPaymentContImpl) UpdateStatus(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	paymentID, err := helpers.ParseUUID(ctx, "payment_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	var request reqOP.UpdatePaymentStatusRequest
	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	if err := cont.OrderPaymentServ.UpdateStatus(accessToken, clientID, paymentID, request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}
