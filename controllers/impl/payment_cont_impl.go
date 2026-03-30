package impl

import (
	"backend/exceptions"
	"backend/helpers"
	"backend/models/domains"
	"backend/models/services"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
)

type PaymentContImpl struct {
	PaymentServ services.PaymentServ
}

func NewPaymentContImpl(paymentServ services.PaymentServ) *PaymentContImpl {
	return &PaymentContImpl{PaymentServ: paymentServ}
}

// ============================================================
// PLATFORM
// ============================================================

// CreatePlatformCheckout @Summary      Create Platform Checkout
// @Description  Buat sesi pembayaran Stripe untuk pembelian plan (platform Aidia), mengembalikan session URL untuk redirect ke halaman pembayaran Stripe
// @Tags         Payment Platform
// @Produce      json
// @Security     BearerAuth
// @Param        plan_id  path      string true "Plan ID"
// @Success      200      {object}  helpers.ApiResponse{data=payment.CheckoutResponse}
// @Failure      400      {object}  helpers.ApiResponse
// @Failure      401      {object}  helpers.ApiResponse
// @Failure      500      {object}  helpers.ApiResponse
// @Router       /payments/platform/checkout/{plan_id} [post]
func (cont *PaymentContImpl) CreatePlatformCheckout(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	planID, err := helpers.ParseUUID(ctx, "plan_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, errServ := cont.PaymentServ.CreatePlatformCheckout(accessToken, planID)
	if errServ != nil {
		exceptions.ErrorHandler(ctx, errServ)
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

// CreatePaymentFromExisting @Summary      Create Payment From Existing Invoice
// @Description  Buat ulang sesi pembayaran Stripe untuk invoice yang belum dibayar, digunakan jika sesi sebelumnya expired
// @Tags         Payment Platform
// @Produce      json
// @Security     BearerAuth
// @Param        invoice_id  path      string true "Invoice ID"
// @Success      200         {object}  helpers.ApiResponse{data=payment.CheckoutResponse}
// @Failure      400         {object}  helpers.ApiResponse
// @Failure      401         {object}  helpers.ApiResponse
// @Failure      500         {object}  helpers.ApiResponse
// @Router       /payments/platform/invoices/{invoice_id}/pay [post]
func (cont *PaymentContImpl) CreatePaymentFromExisting(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	invoiceID, err := helpers.ParseUUID(ctx, "invoice_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, errServ := cont.PaymentServ.CreatePaymentFromExisting(accessToken, invoiceID)
	if errServ != nil {
		exceptions.ErrorHandler(ctx, errServ)
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

// GetPlatformInvoices @Summary      Get Platform Invoices
// @Description  Ambil semua invoice pembelian plan milik tenant yang sedang login dengan pagination
// @Tags         Payment Platform
// @Produce      json
// @Security     BearerAuth
// @Param        page   query     int false "Page"
// @Param        limit  query     int false "Limit"
// @Success      200    {object}  helpers.ApiResponse{data=pagination.Response}
// @Failure      401    {object}  helpers.ApiResponse
// @Failure      500    {object}  helpers.ApiResponse
// @Router       /payments/platform/invoices [get]
func (cont *PaymentContImpl) GetPlatformInvoices(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)
	pg := domains.ParsePagination(ctx)

	result, errServ := cont.PaymentServ.GetPlatformInvoices(accessToken, pg)
	if errServ != nil {
		exceptions.ErrorHandler(ctx, errServ)
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

// GetPlatformInvoiceByID @Summary      Get Platform Invoice By ID
// @Description  Ambil detail invoice pembelian plan berdasarkan invoice ID
// @Tags         Payment Platform
// @Produce      json
// @Security     BearerAuth
// @Param        invoice_id  path      string true "Invoice ID"
// @Success      200         {object}  helpers.ApiResponse{data=payment.InvoiceResponse}
// @Failure      400         {object}  helpers.ApiResponse
// @Failure      401         {object}  helpers.ApiResponse
// @Failure      500         {object}  helpers.ApiResponse
// @Router       /payments/platform/invoices/{invoice_id} [get]
func (cont *PaymentContImpl) GetPlatformInvoiceByID(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	invoiceID, err := helpers.ParseUUID(ctx, "invoice_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, errServ := cont.PaymentServ.GetPlatformInvoiceByID(accessToken, invoiceID)
	if errServ != nil {
		exceptions.ErrorHandler(ctx, errServ)
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

// HandlePlatformWebhook @Summary      Handle Platform Webhook
// @Description  Endpoint untuk menerima event webhook dari Stripe platform Aidia (invoice.paid / invoice.payment_failed), tidak memerlukan autentikasi (TIDAK DIPAKAI KARENA INI UNTUK WEBHOOK SAJA)
// @Tags         Payment Platform
// @Accept       json
// @Produce      json
// @Param        Stripe-Signature  header    string true "Stripe Webhook Signature"
// @Success      200               {object}  helpers.ApiResponse
// @Failure      400               {object}  helpers.ApiResponse
// @Failure      500               {object}  helpers.ApiResponse
// @Router       /payments/platform/webhook [post]
func (cont *PaymentContImpl) HandlePlatformWebhook(ctx *gin.Context) {
	payload, err := ctx.GetRawData()

	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	signature := ctx.GetHeader("Stripe-Signature")
	if signature == "" {
		exceptions.ErrorHandler(ctx, fmt.Errorf("missing stripe signature"))
		return
	}

	errServ := cont.PaymentServ.HandlePlatformWebhook(payload, signature)
	if errServ != nil {
		exceptions.ErrorHandler(ctx, errServ)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    nil,
	}

	errResponse := helpers.WriteToResponseBody(ctx, response.Code, response)
	if errResponse != nil {
		exceptions.ErrorHandler(ctx, errResponse)
		return
	}
}

// ============================================================
// CLIENT
// ============================================================

// CreateClientCheckout @Summary      [BELUM DIGUNAKAN] Create Client Checkout
// @Description  [BELUM DIGUNAKAN] Buat sesi pembayaran Stripe untuk order milik tenant (Stripe per tenant). Endpoint ini belum aktif digunakan karena fitur pembayaran order tenant masih dalam pengembangan.
// @Tags         Payment Client
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string true "Client ID (tenant UUID)"
// @Param        order_id   path      string true "Order ID"
// @Success      200       {object}  helpers.ApiResponse{data=payment.CheckoutResponse}
// @Failure      400       {object}  helpers.ApiResponse
// @Failure      401       {object}  helpers.ApiResponse
// @Failure      500       {object}  helpers.ApiResponse
// @Router       /payments/client/{client_id}/checkout/{order_id} [post]
func (cont *PaymentContImpl) CreateClientCheckout(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	orderID, err := helpers.ParseUUID(ctx, "order_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, errServ := cont.PaymentServ.CreateClientCheckout(clientID, orderID)
	if errServ != nil {
		exceptions.ErrorHandler(ctx, errServ)
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

// GetClientInvoices @Summary      [BELUM DIGUNAKAN] Get Client Invoices
// @Description  [BELUM DIGUNAKAN] Ambil semua invoice order milik tenant yang sedang login dengan pagination. Endpoint ini belum aktif digunakan karena fitur pembayaran order tenant masih dalam pengembangan.
// @Tags         Payment Client
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string true "Client ID (tenant UUID)"
// @Param        page       query     int     false "Page"
// @Param        limit      query     int     false "Limit"
// @Success      200        {object}  helpers.ApiResponse{data=pagination.Response}
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /payments/client/{client_id}/invoices [get]
func (cont *PaymentContImpl) GetClientInvoices(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	pg := domains.ParsePagination(ctx)

	result, errServ := cont.PaymentServ.GetClientInvoices(clientID, pg)
	if errServ != nil {
		exceptions.ErrorHandler(ctx, errServ)
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

// HandleClientWebhook @Summary      [BELUM DIGUNAKAN] Handle Client Webhook
// @Description  [BELUM DIGUNAKAN] Endpoint untuk menerima event webhook dari Stripe per tenant. Endpoint ini belum aktif digunakan karena fitur pembayaran order tenant masih dalam pengembangan.
// @Tags         Payment Client
// @Accept       json
// @Produce      json
// @Param        schema            path      string true "Tenant Schema"
// @Param        Stripe-Signature  header    string true "Stripe Webhook Signature"
// @Success      200               {object}  helpers.ApiResponse
// @Failure      400               {object}  helpers.ApiResponse
// @Failure      500               {object}  helpers.ApiResponse
// @Router       /payments/client/webhook/{schema} [post]
func (cont *PaymentContImpl) HandleClientWebhook(ctx *gin.Context) {
	schema := ctx.Param("schema")
	if schema == "" {
		exceptions.ErrorHandler(ctx, fmt.Errorf("missing schema param"))
		return
	}

	payload, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	signature := ctx.GetHeader("Stripe-Signature")
	if signature == "" {
		exceptions.ErrorHandler(ctx, fmt.Errorf("missing stripe signature"))
		return
	}

	errServ := cont.PaymentServ.HandleClientWebhook(schema, payload, signature)
	if errServ != nil {
		exceptions.ErrorHandler(ctx, errServ)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    nil,
	}

	errResponse := helpers.WriteToResponseBody(ctx, response.Code, response)
	if errResponse != nil {
		exceptions.ErrorHandler(ctx, errResponse)
		return
	}
}
