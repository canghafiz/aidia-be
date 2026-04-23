package impl

import (
	"backend/exceptions"
	"backend/helpers"
	"backend/models/domains"
	"backend/models/services"
	"fmt"
	"io"
	"net/http"

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

// GetAvailableGateways godoc
// @Summary      Get Available Payment Gateways
// @Description  Returns a list of payment gateways that are configured and available for platform checkout (e.g. stripe, hitpay)
// @Tags         Payment Platform
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  helpers.ApiResponse{data=[]string}
// @Failure      401  {object}  helpers.ApiResponse
// @Router       /payments/platform/gateways [get]
func (cont *PaymentContImpl) GetAvailableGateways(ctx *gin.Context) {
	gateways := cont.PaymentServ.GetAvailableGateways()

	errResponse := helpers.WriteToResponseBody(ctx, http.StatusOK, helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    gateways,
	})
	if errResponse != nil {
		exceptions.ErrorHandler(ctx, errResponse)
	}
}

// CreatePlatformCheckout @Summary      Create Platform Checkout
// @Description  Buat sesi pembayaran untuk pembelian plan (platform Aidia). Gunakan query param `gateway` untuk memilih gateway (stripe / hitpay). Jika kosong, pakai gateway default dari konfigurasi.
// @Tags         Payment Platform
// @Produce      json
// @Security     BearerAuth
// @Param        plan_id  path      string true  "Plan ID"
// @Param        gateway  query     string false "Payment gateway: stripe | hitpay (default: active gateway)"
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

	gateway := ctx.Query("gateway")

	result, errServ := cont.PaymentServ.CreatePlatformCheckout(accessToken, planID, gateway)
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
// @Description  Buat ulang sesi pembayaran untuk invoice yang belum dibayar. Gunakan query param `gateway` untuk memilih gateway (stripe / hitpay).
// @Tags         Payment Platform
// @Produce      json
// @Security     BearerAuth
// @Param        invoice_id  path      string true  "Invoice ID"
// @Param        gateway     query     string false "Payment gateway: stripe | hitpay (default: active gateway)"
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

	gateway := ctx.Query("gateway")

	result, errServ := cont.PaymentServ.CreatePaymentFromExisting(accessToken, invoiceID, gateway)
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

// HandlePlatformWebhookStripe godoc
// @Summary      Handle Platform Stripe Webhook
// @Description  Receives Stripe webhook events for Aidia platform payments (invoice.paid / invoice.payment_failed). No authentication required — validated via Stripe-Signature header.
// @Tags         Payment Platform
// @Accept       json
// @Produce      json
// @Param        Stripe-Signature  header  string  true  "Stripe Webhook Signature"
// @Success      200  {object}  helpers.ApiResponse
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      500  {object}  helpers.ApiResponse
// @Router       /payments/platform/webhook/stripe [post]
func (cont *PaymentContImpl) HandlePlatformWebhookStripe(ctx *gin.Context) {
	payload, err := ctx.GetRawData()
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	signature := ctx.GetHeader("Stripe-Signature")
	if signature == "" {
		exceptions.ErrorHandler(ctx, fmt.Errorf("missing Stripe-Signature header"))
		return
	}

	if errServ := cont.PaymentServ.HandlePlatformWebhookStripe(payload, signature); errServ != nil {
		exceptions.ErrorHandler(ctx, errServ)
		return
	}

	errResponse := helpers.WriteToResponseBody(ctx, http.StatusOK, helpers.ApiResponse{Success: true, Code: 200})
	if errResponse != nil {
		exceptions.ErrorHandler(ctx, errResponse)
	}
}

// HandlePlatformWebhookHitPay godoc
// @Summary      Handle Platform HitPay Webhook
// @Description  Receives HitPay webhook events for Aidia platform payments (status: completed / failed). No authentication required — validated via HMAC-SHA256 in the form body.
// @Tags         Payment Platform
// @Accept       application/x-www-form-urlencoded
// @Produce      json
// @Param        payment_id        formData  string  true  "HitPay Payment ID"
// @Param        payment_request_id formData  string  true  "HitPay Payment Request ID"
// @Param        status            formData  string  true  "Payment status (completed / failed)"
// @Param        reference_number  formData  string  false "Reference number (invoice number)"
// @Param        amount            formData  string  false "Payment amount"
// @Param        currency          formData  string  false "Currency"
// @Param        hmac              formData  string  true  "HMAC-SHA256 signature"
// @Success      200  {object}  helpers.ApiResponse
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      500  {object}  helpers.ApiResponse
// @Router       /payments/platform/webhook/hitpay [post]
func (cont *PaymentContImpl) HandlePlatformWebhookHitPay(ctx *gin.Context) {
	if err := ctx.Request.ParseForm(); err != nil {
		exceptions.ErrorHandler(ctx, fmt.Errorf("failed to parse form: %w", err))
		return
	}

	formValues := make(map[string]string)
	for k, v := range ctx.Request.PostForm {
		if len(v) > 0 {
			formValues[k] = v[0]
		}
	}

	if errServ := cont.PaymentServ.HandlePlatformWebhookHitPay(formValues); errServ != nil {
		exceptions.ErrorHandler(ctx, errServ)
		return
	}

	errResponse := helpers.WriteToResponseBody(ctx, http.StatusOK, helpers.ApiResponse{Success: true, Code: 200})
	if errResponse != nil {
		exceptions.ErrorHandler(ctx, errResponse)
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

// HandleClientWebhookStripe godoc
// @Summary      Handle Client Stripe Webhook
// @Description  Receives Stripe webhook events for a specific tenant's order payments (invoice.paid / invoice.payment_failed). No authentication required — validated via Stripe-Signature header.
// @Tags         Payment Client
// @Accept       json
// @Produce      json
// @Param        schema            path    string  true  "Tenant Schema"
// @Param        Stripe-Signature  header  string  true  "Stripe Webhook Signature"
// @Success      200  {object}  helpers.ApiResponse
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      500  {object}  helpers.ApiResponse
// @Router       /payments/client/{client_id}/webhook/stripe/{schema} [post]
func (cont *PaymentContImpl) HandleClientWebhookStripe(ctx *gin.Context) {
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
		exceptions.ErrorHandler(ctx, fmt.Errorf("missing Stripe-Signature header"))
		return
	}

	if errServ := cont.PaymentServ.HandleClientWebhookStripe(schema, payload, signature); errServ != nil {
		exceptions.ErrorHandler(ctx, errServ)
		return
	}

	errResponse := helpers.WriteToResponseBody(ctx, http.StatusOK, helpers.ApiResponse{Success: true, Code: 200})
	if errResponse != nil {
		exceptions.ErrorHandler(ctx, errResponse)
	}
}

// HandleClientWebhookHitPay godoc
// @Summary      Handle Client HitPay Webhook
// @Description  Receives HitPay webhook events for a specific tenant's order payments (status: completed / failed). No authentication required — validated via HMAC-SHA256 in the form body.
// @Tags         Payment Client
// @Accept       application/x-www-form-urlencoded
// @Produce      json
// @Param        schema             path      string  true  "Tenant Schema"
// @Param        payment_id         formData  string  true  "HitPay Payment ID"
// @Param        payment_request_id formData  string  true  "HitPay Payment Request ID"
// @Param        status             formData  string  true  "Payment status (completed / failed)"
// @Param        reference_number   formData  string  false "Reference number"
// @Param        amount             formData  string  false "Payment amount"
// @Param        currency           formData  string  false "Currency"
// @Param        hmac               formData  string  true  "HMAC-SHA256 signature"
// @Success      200  {object}  helpers.ApiResponse
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      500  {object}  helpers.ApiResponse
// @Router       /payments/client/{client_id}/webhook/hitpay/{schema} [post]
func (cont *PaymentContImpl) HandleClientWebhookHitPay(ctx *gin.Context) {
	schema := ctx.Param("schema")
	if schema == "" {
		exceptions.ErrorHandler(ctx, fmt.Errorf("missing schema param"))
		return
	}

	if err := ctx.Request.ParseForm(); err != nil {
		exceptions.ErrorHandler(ctx, fmt.Errorf("failed to parse form: %w", err))
		return
	}

	formValues := make(map[string]string)
	for k, v := range ctx.Request.PostForm {
		if len(v) > 0 {
			formValues[k] = v[0]
		}
	}

	if errServ := cont.PaymentServ.HandleClientWebhookHitPay(schema, formValues); errServ != nil {
		exceptions.ErrorHandler(ctx, errServ)
		return
	}

	errResponse := helpers.WriteToResponseBody(ctx, http.StatusOK, helpers.ApiResponse{Success: true, Code: 200})
	if errResponse != nil {
		exceptions.ErrorHandler(ctx, errResponse)
	}
}
