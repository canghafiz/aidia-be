package impl

import (
	"backend/exceptions"
	"backend/helpers"
	"backend/models/domains"
	reqOrder "backend/models/requests/order"
	"backend/models/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

type OrderContImpl struct {
	OrderServ services.OrderServ
}

func NewOrderContImpl(orderServ services.OrderServ) *OrderContImpl {
	return &OrderContImpl{OrderServ: orderServ}
}

// Create @Summary      Create Order
// @Description  Buat order baru. Customer dicek berdasarkan phone number — jika sudah ada, data customer lama dipakai. Jika belum ada, customer baru dibuat.
// @Tags         Order
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string                    true "Client ID"
// @Param        request    body      order.CreateOrderRequest  true "Create Order Request"
// @Success      200        {object}  helpers.ApiResponse{data=order.DetailResponse}
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/orders [post]
func (cont *OrderContImpl) Create(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	var request reqOrder.CreateOrderRequest
	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, err := cont.OrderServ.Create(accessToken, clientID, request)
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

// GetAll @Summary      Get All Orders
// @Description  Ambil semua order dengan pagination
// @Tags         Order
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path   string  true  "Client ID"
// @Param        page       query  int     false "Page"
// @Param        limit      query  int     false "Limit"
// @Success      200        {object}  helpers.ApiResponse{data=pagination.Response}
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/orders [get]
func (cont *OrderContImpl) GetAll(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	pg := domains.ParsePagination(ctx)

	result, err := cont.OrderServ.GetAll(accessToken, clientID, pg)
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

// GetByID @Summary      Get Order By ID
// @Description  Ambil detail order beserta customer, produk, dan payment
// @Tags         Order
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string true "Client ID"
// @Param        order_id   path  int    true "Order ID"
// @Success      200        {object}  helpers.ApiResponse{data=order.DetailResponse}
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/orders/{order_id} [get]
func (cont *OrderContImpl) GetByID(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	orderID, err := strconv.Atoi(ctx.Param("order_id"))
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, err := cont.OrderServ.GetByID(accessToken, clientID, orderID)
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

// UpdateStatus @Summary      Update Order Status
// @Description  Update status order (Pending/Confirmed/Completed/Cancelled)
// @Tags         Order
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string                         true "Client ID"
// @Param        order_id   path  int                            true "Order ID"
// @Param        request    body  order.UpdateOrderStatusRequest true "Update Order Status Request"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/orders/{order_id}/status [patch]
func (cont *OrderContImpl) UpdateStatus(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	orderID, err := strconv.Atoi(ctx.Param("order_id"))
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	var request reqOrder.UpdateOrderStatusRequest
	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	if err := cont.OrderServ.UpdateStatus(accessToken, clientID, orderID, request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}
