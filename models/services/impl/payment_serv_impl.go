package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	"backend/models/responses/pagination"
	paymentRes "backend/models/responses/payment"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	stripecustomer "github.com/stripe/stripe-go/v81/customer"
	stripeinvoice "github.com/stripe/stripe-go/v81/invoice"
	stripeinvoiceitem "github.com/stripe/stripe-go/v81/invoiceitem"
	"github.com/stripe/stripe-go/v81/webhook"
	"gorm.io/gorm"
)

type PaymentServImpl struct {
	Db               *gorm.DB
	JwtKey           string
	UserRepo         repositories.UsersRepo
	TenantPlanRepo   repositories.TenantPlanRepo
	PlanRepo         repositories.PlanRepo
	TenantRepo       repositories.TenantRepo
	SettingRepo      repositories.SettingRepo
	OrderPaymentRepo repositories.OrderPaymentRepo
	OrderRepo        repositories.OrderRepo
}

func NewPaymentServImpl(
	db *gorm.DB,
	jwtKey string,
	userRepo repositories.UsersRepo,
	tenantPlanRepo repositories.TenantPlanRepo,
	planRepo repositories.PlanRepo,
	tenantRepo repositories.TenantRepo,
	settingRepo repositories.SettingRepo,
	orderPaymentRepo repositories.OrderPaymentRepo,
	orderRepo repositories.OrderRepo,
) *PaymentServImpl {
	return &PaymentServImpl{
		Db:               db,
		JwtKey:           jwtKey,
		UserRepo:         userRepo,
		TenantPlanRepo:   tenantPlanRepo,
		PlanRepo:         planRepo,
		TenantRepo:       tenantRepo,
		SettingRepo:      settingRepo,
		OrderPaymentRepo: orderPaymentRepo,
		OrderRepo:        orderRepo,
	}
}

// ============================================================
// HELPERS
// ============================================================

func (serv *PaymentServImpl) getKey(schema, subGroupName, name string) (string, error) {
	settings, err := serv.SettingRepo.GetByGroupAndSubGroupName(serv.Db, schema, "integration", subGroupName)
	if err != nil || len(settings) == 0 {
		return "", fmt.Errorf("integration config not found for sub group: %s", subGroupName)
	}
	for _, s := range settings {
		if s.Name == name {
			return s.Value, nil
		}
	}
	return "", fmt.Errorf("key not found: %s / %s", subGroupName, name)
}

// activeGateway returns "stripe" or "hitpay" based on the public.setting value.
// Falls back to "stripe" if the setting is missing.
func (serv *PaymentServImpl) activeGateway() string {
	settings, err := serv.SettingRepo.GetByGroupAndSubGroupName(serv.Db, "public", "integration", "Payment Gateway")
	if err != nil || len(settings) == 0 {
		return "stripe"
	}
	for _, s := range settings {
		if s.Name == "active-payment-gateway" {
			return s.Value
		}
	}
	return "stripe"
}

func (serv *PaymentServImpl) getTenantByToken(accessToken string) (*domains.Tenant, error) {
	userIDStr, err := helpers.GetUserIdFromToken(accessToken, serv.JwtKey)
	if err != nil {
		return nil, err
	}
	userID, _ := uuid.Parse(*userIDStr)

	tenant, err := serv.TenantRepo.GetByUserID(serv.Db, userID)
	if err != nil {
		return nil, fmt.Errorf("tenant profile not found, please complete registration")
	}
	return tenant, nil
}

// buildStripeInvoice creates a Stripe Customer → InvoiceItem → Invoice
// and returns (invoiceID, hostedURL, error).
func (serv *PaymentServImpl) buildStripeInvoice(
	tenantName, tenantEmail, invoiceNumber, planName, duration string,
	priceInCents int64,
	tenantPlanID uuid.UUID,
) (string, string, error) {
	customerParams := &stripe.CustomerParams{
		Name:  stripe.String(tenantName),
		Email: stripe.String(tenantEmail),
		Metadata: map[string]string{
			"tenant_plan_id": tenantPlanID.String(),
			"invoice_number": invoiceNumber,
		},
	}
	customer, err := stripecustomer.New(customerParams)
	if err != nil {
		return "", "", fmt.Errorf("failed to create stripe customer: %w", err)
	}

	invoiceParams := &stripe.InvoiceParams{
		Customer:         stripe.String(customer.ID),
		CollectionMethod: stripe.String("send_invoice"),
		DaysUntilDue:     stripe.Int64(7),
		Currency:         stripe.String("sgd"),
		Metadata: map[string]string{
			"tenant_plan_id": tenantPlanID.String(),
			"invoice_number": invoiceNumber,
		},
	}
	inv, err := stripeinvoice.New(invoiceParams)
	if err != nil {
		return "", "", fmt.Errorf("failed to create stripe invoice: %w", err)
	}

	itemParams := &stripe.InvoiceItemParams{
		Customer:    stripe.String(customer.ID),
		Invoice:     stripe.String(inv.ID),
		Amount:      stripe.Int64(priceInCents),
		Currency:    stripe.String("sgd"),
		Description: stripe.String(fmt.Sprintf("%s | Invoice: %s | Duration: %s", planName, invoiceNumber, duration)),
	}
	if _, err = stripeinvoiceitem.New(itemParams); err != nil {
		return "", "", fmt.Errorf("failed to create stripe invoice item: %w", err)
	}

	finalInv, err := stripeinvoice.FinalizeInvoice(inv.ID, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to finalize stripe invoice: %w", err)
	}
	log.Printf("[buildStripeInvoice] finalized: %s, amount: %d, url: %s", finalInv.ID, finalInv.AmountDue, finalInv.HostedInvoiceURL)

	return finalInv.ID, finalInv.HostedInvoiceURL, nil
}

// buildHitPayPayment creates a HitPay payment request and returns (paymentID, paymentURL, error).
func (serv *PaymentServImpl) buildHitPayPayment(
	apiKey string,
	tenantName, tenantEmail, invoiceNumber, planName, duration string,
	price float64,
	tenantPlanID uuid.UUID,
	sandbox bool,
) (string, string, error) {
	redirectURL := os.Getenv("HITPAY_REDIRECT_URL")
	webhookURL := os.Getenv("HITPAY_PLATFORM_WEBHOOK_URL")

	if redirectURL == "" {
		return "", "", fmt.Errorf("HITPAY_REDIRECT_URL env not set")
	}
	if webhookURL == "" {
		return "", "", fmt.Errorf("HITPAY_PLATFORM_WEBHOOK_URL env not set")
	}

	resp, err := helpers.HitPayCreatePayment(apiKey, helpers.HitPayPaymentRequest{
		Amount:          fmt.Sprintf("%.2f", price),
		Currency:        "SGD",
		Email:           tenantEmail,
		Name:            tenantName,
		ReferenceNumber: invoiceNumber,
		RedirectURL:     redirectURL,
		WebhookURL:      webhookURL,
		Purpose:         fmt.Sprintf("%s | %s", planName, duration),
	}, sandbox)
	if err != nil {
		return "", "", fmt.Errorf("hitpay payment creation failed: %w", err)
	}

	log.Printf("[buildHitPayPayment] created: id=%s, url=%s", resp.ID, resp.URL)
	return resp.ID, resp.URL, nil
}

// ============================================================
// PLATFORM — tenant purchases a plan
// ============================================================

// GetAvailableGateways returns gateway keys that have a non-empty API key configured.
func (serv *PaymentServImpl) GetAvailableGateways() []string {
	var available []string

	stripeKey, err := serv.getKey("public", "Stripe Aidia", "stripe-aidia-secret-key")
	if err == nil && stripeKey != "" && stripeKey != "{stripe-aidia-secret-key}" {
		available = append(available, "stripe")
	}

	hitpayKey, err := serv.getKey("public", "HitPay Aidia", "hitpay-aidia-api-key")
	if err == nil && hitpayKey != "" {
		available = append(available, "hitpay")
	}

	if len(available) == 0 {
		available = append(available, "stripe") // safe fallback
	}
	return available
}

func (serv *PaymentServImpl) CreatePlatformCheckout(accessToken string, planID uuid.UUID, gateway string) (*paymentRes.CheckoutResponse, error) {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"Client"})
	if err != nil || !ok {
		return nil, err
	}

	tenant, err := serv.getTenantByToken(accessToken)
	if err != nil {
		return nil, err
	}

	plan, err := serv.PlanRepo.GetByIdPublic(serv.Db, planID)
	if err != nil {
		log.Printf("[PlanRepo].GetByIdPublic error: %v", err)
		return nil, fmt.Errorf("plan not found")
	}

	tx := serv.Db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to start transaction")
	}

	sequence, err := serv.TenantPlanRepo.GetLastSequenceToday(tx)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to generate invoice number")
	}
	invoiceNumber := helpers.GenerateInvoiceNumber(sequence + 1)
	dueDate := time.Now().Add(24 * time.Hour)

	tenantPlan, err := serv.TenantPlanRepo.Create(tx, domains.TenantPlan{
		TenantID:       tenant.TenantID,
		PlanID:         planID,
		InvoiceNumber:  invoiceNumber,
		Duration:       plan.Duration,
		IsMonth:        plan.IsMonth,
		Price:          plan.Price,
		PaymentDueDate: &dueDate,
		PlanStatus:     "Inactive",
		IsPaid:         false,
	})
	if err != nil {
		tx.Rollback()
		log.Printf("[TenantPlanRepo].Create error: %v", err)
		return nil, fmt.Errorf("failed to create invoice")
	}

	if gateway == "" {
		gateway = serv.activeGateway()
	}
	tenantName := tenant.User.Name
	tenantEmail := tenant.User.Email
	duration := paymentRes.FormatDuration(plan.Duration, plan.IsMonth)

	var sessionID, sessionURL string

	switch gateway {
	case "hitpay":
		apiKey, err := serv.getKey("public", "HitPay Aidia", "hitpay-aidia-api-key")
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("HitPay API key not configured")
		}
		sandboxVal, _ := serv.getKey("public", "HitPay Aidia", "hitpay-aidia-sandbox")
		sandbox := sandboxVal == "true"

		sessionID, sessionURL, err = serv.buildHitPayPayment(
			apiKey, tenantName, tenantEmail, invoiceNumber, plan.Name, duration,
			plan.Price, tenantPlan.ID, sandbox,
		)
		if err != nil {
			tx.Rollback()
			log.Printf("[HitPay].buildHitPayPayment error: %v", err)
			return nil, fmt.Errorf("failed to create payment")
		}

	case "stripe":
		secretKey, err := serv.getKey("public", "Stripe Aidia", "stripe-aidia-secret-key")
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("Stripe secret key not configured")
		}
		stripe.Key = secretKey
		priceInCents := int64(plan.Price * 100)

		sessionID, sessionURL, err = serv.buildStripeInvoice(
			tenantName, tenantEmail, invoiceNumber, plan.Name, duration,
			priceInCents, tenantPlan.ID,
		)
		if err != nil {
			tx.Rollback()
			log.Printf("[Stripe].buildStripeInvoice error: %v", err)
			return nil, fmt.Errorf("failed to create payment invoice")
		}

	default:
		tx.Rollback()
		return nil, fmt.Errorf("unsupported payment gateway: %s", gateway)
	}

	pending := "pending"
	if err := serv.TenantPlanRepo.UpdatePaymentSession(tx, domains.TenantPlan{
		ID:                    tenantPlan.ID,
		PaymentGateway:        gateway,
		PaymentSessionID:      &sessionID,
		PaymentSessionURL:     &sessionURL,
		SubscriptionInvoiceID: &sessionID,
		PaymentGatewayStatus:  &pending,
	}); err != nil {
		tx.Rollback()
		log.Printf("[TenantPlanRepo].UpdatePaymentSession error: %v", err)
		return nil, fmt.Errorf("failed to update session")
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to commit transaction")
	}

	return &paymentRes.CheckoutResponse{
		InvoiceID:  tenantPlan.ID,
		SessionID:  sessionID,
		SessionURL: sessionURL,
	}, nil
}

func (serv *PaymentServImpl) CreatePaymentFromExisting(accessToken string, tenantPlanID uuid.UUID, gateway string) (*paymentRes.CheckoutResponse, error) {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"Client"})
	if err != nil || !ok {
		return nil, err
	}

	tenant, err := serv.getTenantByToken(accessToken)
	if err != nil {
		return nil, err
	}

	tenantPlan, err := serv.TenantPlanRepo.GetByIDAndTenantID(serv.Db, tenantPlanID, tenant.TenantID)
	if err != nil {
		log.Printf("[TenantPlanRepo].GetByIDAndTenantID error: %v", err)
		return nil, fmt.Errorf("invoice not found or already paid")
	}

	if tenantPlan.ExpiredDate != nil && tenantPlan.ExpiredDate.Before(time.Now()) {
		return nil, fmt.Errorf("invoice already expired")
	}
	if tenantPlan.PaymentDueDate != nil && tenantPlan.PaymentDueDate.Before(time.Now()) {
		return nil, fmt.Errorf("payment due date has passed, please create a new invoice")
	}

	if gateway == "" {
		gateway = serv.activeGateway()
	}
	tenantName := tenant.User.Name
	tenantEmail := tenant.User.Email
	duration := paymentRes.FormatDuration(tenantPlan.Duration, tenantPlan.IsMonth)

	var sessionID, sessionURL string

	switch gateway {
	case "hitpay":
		apiKey, err := serv.getKey("public", "HitPay Aidia", "hitpay-aidia-api-key")
		if err != nil {
			return nil, fmt.Errorf("HitPay API key not configured")
		}
		sandboxVal, _ := serv.getKey("public", "HitPay Aidia", "hitpay-aidia-sandbox")
		sandbox := sandboxVal == "true"

		sessionID, sessionURL, err = serv.buildHitPayPayment(
			apiKey, tenantName, tenantEmail, tenantPlan.InvoiceNumber,
			tenantPlan.Plan.Name, duration, tenantPlan.Price, tenantPlan.ID, sandbox,
		)
		if err != nil {
			log.Printf("[HitPay].buildHitPayPayment error: %v", err)
			return nil, fmt.Errorf("failed to create payment")
		}

	case "stripe":
		secretKey, err := serv.getKey("public", "Stripe Aidia", "stripe-aidia-secret-key")
		if err != nil {
			return nil, fmt.Errorf("Stripe secret key not configured")
		}
		stripe.Key = secretKey
		priceInCents := int64(tenantPlan.Price * 100)

		sessionID, sessionURL, err = serv.buildStripeInvoice(
			tenantName, tenantEmail, tenantPlan.InvoiceNumber,
			tenantPlan.Plan.Name, duration, priceInCents, tenantPlan.ID,
		)
		if err != nil {
			log.Printf("[Stripe].buildStripeInvoice error: %v", err)
			return nil, fmt.Errorf("failed to create payment invoice")
		}

	default:
		return nil, fmt.Errorf("unsupported payment gateway: %s", gateway)
	}

	pending := "pending"
	if err := serv.TenantPlanRepo.UpdatePaymentSession(serv.Db, domains.TenantPlan{
		ID:                    tenantPlan.ID,
		PaymentGateway:        gateway,
		PaymentSessionID:      &sessionID,
		PaymentSessionURL:     &sessionURL,
		SubscriptionInvoiceID: &sessionID,
		PaymentGatewayStatus:  &pending,
	}); err != nil {
		log.Printf("[TenantPlanRepo].UpdatePaymentSession error: %v", err)
		return nil, fmt.Errorf("failed to update session")
	}

	return &paymentRes.CheckoutResponse{
		InvoiceID:  tenantPlan.ID,
		SessionID:  sessionID,
		SessionURL: sessionURL,
	}, nil
}

func (serv *PaymentServImpl) GetPlatformInvoices(accessToken string, pg domains.Pagination) (*pagination.Response, error) {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"Client"})
	if err != nil || !ok {
		return nil, err
	}

	tenant, err := serv.getTenantByToken(accessToken)
	if err != nil {
		return nil, err
	}

	plans, total, err := serv.TenantPlanRepo.GetByTenantID(serv.Db, tenant.TenantID, pg)
	if err != nil {
		log.Printf("[TenantPlanRepo].GetByTenantID error: %v", err)
		return nil, fmt.Errorf("failed to get invoices")
	}

	responses := paymentRes.ToInvoiceResponses(plans)
	result := pagination.ToResponse(responses, total, pg.Page, pg.Limit)
	return &result, nil
}

func (serv *PaymentServImpl) GetPlatformInvoiceByID(accessToken string, id uuid.UUID) (*paymentRes.InvoiceResponse, error) {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"Client", "SuperAdmin"})
	if err != nil || !ok {
		return nil, err
	}

	tenantPlan, err := serv.TenantPlanRepo.GetByID(serv.Db, id)
	if err != nil {
		log.Printf("[TenantPlanRepo].GetByID error: %v", err)
		return nil, fmt.Errorf("invoice not found")
	}

	response := paymentRes.ToInvoiceResponse(*tenantPlan)
	return &response, nil
}

// ============================================================
// PLATFORM WEBHOOKS
// ============================================================

func (serv *PaymentServImpl) HandlePlatformWebhookStripe(payload []byte, signature string) error {
	webhookSecret, err := serv.getKey("public", "Stripe Aidia", "stripe-aidia-webhook-secret")
	if err != nil {
		return err
	}

	event, errEvent := webhook.ConstructEventWithOptions(payload, signature, webhookSecret,
		webhook.ConstructEventOptions{
			Tolerance:                300 * time.Second,
			IgnoreAPIVersionMismatch: true,
		},
	)
	if errEvent != nil {
		log.Printf("[StripeWebhook] ConstructEvent error: %v", errEvent)
		return fmt.Errorf("invalid webhook signature")
	}

	switch event.Type {
	case "invoice.paid":
		invoiceID, ok := event.Data.Object["id"].(string)
		if !ok {
			return fmt.Errorf("invalid invoice data")
		}
		log.Printf("[StripeWebhook] invoice paid: %s", invoiceID)
		return serv.markPlatformPaid(invoiceID)

	case "invoice.payment_failed":
		invoiceID, ok := event.Data.Object["id"].(string)
		if !ok {
			return fmt.Errorf("invalid invoice data")
		}
		return serv.markPlatformFailed(invoiceID)
	}

	return nil
}

func (serv *PaymentServImpl) HandlePlatformWebhookHitPay(formValues map[string]string) error {
	salt, err := serv.getKey("public", "HitPay Aidia", "hitpay-aidia-webhook-salt")
	if err != nil {
		return err
	}

	if !helpers.HitPayVerifyWebhook(formValues, salt) {
		return fmt.Errorf("invalid hitpay webhook signature")
	}

	status := formValues["status"]
	referenceNumber := formValues["reference_number"]
	paymentID := formValues["payment_id"]

	log.Printf("[HitPayWebhook] status=%s reference=%s payment_id=%s", status, referenceNumber, paymentID)

	switch status {
	case "completed":
		tenantPlan, err := serv.TenantPlanRepo.GetByPaymentInvoiceID(serv.Db, paymentID)
		if err != nil {
			// Try lookup by reference_number (invoice number) as fallback
			log.Printf("[HitPayWebhook] GetByPaymentInvoiceID failed, trying reference: %v", err)
			return fmt.Errorf("tenant plan not found for payment_id: %s", paymentID)
		}
		return serv.markPlatformPaid(tenantPlan.SubscriptionInvoiceID)

	case "failed":
		tenantPlan, err := serv.TenantPlanRepo.GetByPaymentInvoiceID(serv.Db, paymentID)
		if err != nil {
			return fmt.Errorf("tenant plan not found for payment_id: %s", paymentID)
		}
		return serv.markPlatformFailed(tenantPlan.SubscriptionInvoiceID)
	}

	return nil
}

// markPlatformPaid activates the tenant plan identified by invoiceID.
func (serv *PaymentServImpl) markPlatformPaid(invoiceID interface{}) error {
	var id string
	switch v := invoiceID.(type) {
	case string:
		id = v
	case *string:
		if v == nil {
			return fmt.Errorf("invoiceID is nil")
		}
		id = *v
	default:
		return fmt.Errorf("unsupported invoiceID type")
	}

	tenantPlan, err := serv.TenantPlanRepo.GetByPaymentInvoiceID(serv.Db, id)
	if err != nil {
		log.Printf("[markPlatformPaid] GetByPaymentInvoiceID error: %v, id: %s", err, id)
		return fmt.Errorf("tenant plan not found")
	}
	if tenantPlan.IsPaid {
		return nil // idempotent
	}

	now := time.Now()
	var expiredDate time.Time
	if tenantPlan.IsMonth {
		expiredDate = now.AddDate(0, tenantPlan.Duration, 0)
	} else {
		expiredDate = now.AddDate(tenantPlan.Duration, 0, 0)
	}

	status := "paid"
	if err := serv.TenantPlanRepo.UpdatePaymentStatus(serv.Db, domains.TenantPlan{
		ID:                   tenantPlan.ID,
		IsPaid:               true,
		PlanStatus:           "Active",
		PaymentGatewayStatus: &status,
		PaidAt:               &now,
		StartDate:            &now,
		ExpiredDate:          &expiredDate,
	}); err != nil {
		log.Printf("[markPlatformPaid] UpdatePaymentStatus error: %v", err)
		return fmt.Errorf("failed to update payment status")
	}
	return nil
}

func (serv *PaymentServImpl) markPlatformFailed(invoiceID interface{}) error {
	var id string
	switch v := invoiceID.(type) {
	case string:
		id = v
	case *string:
		if v == nil {
			return fmt.Errorf("invoiceID is nil")
		}
		id = *v
	default:
		return fmt.Errorf("unsupported invoiceID type")
	}

	tenantPlan, err := serv.TenantPlanRepo.GetByPaymentInvoiceID(serv.Db, id)
	if err != nil {
		return fmt.Errorf("tenant plan not found")
	}
	if tenantPlan.IsPaid {
		return nil
	}

	status := "failed"
	msg := "Payment failed"
	if err := serv.TenantPlanRepo.UpdatePaymentStatus(serv.Db, domains.TenantPlan{
		ID:                    tenantPlan.ID,
		IsPaid:                false,
		PlanStatus:            "Inactive",
		PaymentGatewayStatus:  &status,
		PaymentGatewayMessage: &msg,
	}); err != nil {
		log.Printf("[markPlatformFailed] UpdatePaymentStatus error: %v", err)
	}
	return nil
}

// ============================================================
// CLIENT — tenant receives payments from their customers
// ============================================================

func (serv *PaymentServImpl) CreateClientCheckout(clientID uuid.UUID, orderID uuid.UUID) (*paymentRes.CheckoutResponse, error) {
	schema, err := helpers.GetSchema(serv.Db, serv.UserRepo, clientID)
	if err != nil {
		return nil, err
	}

	successURL := os.Getenv("CLIENT_PAYMENT_SUCCESS_URL")
	cancelURL := os.Getenv("CLIENT_PAYMENT_CANCEL_URL")
	if successURL == "" {
		return nil, fmt.Errorf("CLIENT_PAYMENT_SUCCESS_URL env not set")
	}
	if cancelURL == "" {
		return nil, fmt.Errorf("CLIENT_PAYMENT_CANCEL_URL env not set")
	}

	gateway := serv.activeGateway()
	switch gateway {
	case "hitpay":
		if _, err := serv.getKey(schema, "HitPay Client", "hitpay-client-api-key"); err != nil {
			return nil, fmt.Errorf("HitPay client API key not configured")
		}
	default:
		if _, err := serv.getKey(schema, "Stripe Client", "stripe-client-secret-key"); err != nil {
			return nil, fmt.Errorf("Stripe client secret key not configured")
		}
	}

	// TODO: implement full order payment flow per gateway
	return &paymentRes.CheckoutResponse{
		InvoiceID:  orderID,
		SessionID:  "",
		SessionURL: "",
	}, nil
}

func (serv *PaymentServImpl) GetClientInvoices(clientID uuid.UUID, pg domains.Pagination) (*pagination.Response, error) {
	return &pagination.Response{}, nil
}

func (serv *PaymentServImpl) HandleClientWebhookStripe(schema string, payload []byte, signature string) error {
	webhookSecret, err := serv.getKey(schema, "Stripe Client", "stripe-client-webhook-secret")
	if err != nil {
		return err
	}

	event, errEvent := webhook.ConstructEventWithOptions(payload, signature, webhookSecret,
		webhook.ConstructEventOptions{
			Tolerance:                300 * time.Second,
			IgnoreAPIVersionMismatch: true,
		},
	)
	if errEvent != nil {
		log.Printf("[ClientStripeWebhook] ConstructEvent error: %v", errEvent)
		return fmt.Errorf("invalid webhook signature")
	}

	switch event.Type {
	case "invoice.paid":
		invoiceID, ok := event.Data.Object["id"].(string)
		if !ok {
			return fmt.Errorf("invalid invoice data")
		}
		log.Printf("[ClientStripeWebhook] Payment success schema=%s invoice=%s", schema, invoiceID)
		return serv.markClientPaid(schema, invoiceID)

	case "invoice.payment_failed":
		invoiceID, ok := event.Data.Object["id"].(string)
		if !ok {
			return fmt.Errorf("invalid invoice data")
		}
		log.Printf("[ClientStripeWebhook] Payment failed schema=%s invoice=%s", schema, invoiceID)
		return serv.markClientFailed(schema, invoiceID)
	}

	return nil
}

func (serv *PaymentServImpl) HandleClientWebhookHitPay(schema string, formValues map[string]string) error {
	salt, err := serv.getKey(schema, "HitPay Client", "hitpay-client-webhook-salt")
	if err != nil {
		return err
	}

	if !helpers.HitPayVerifyWebhook(formValues, salt) {
		return fmt.Errorf("invalid hitpay webhook signature")
	}

	status := formValues["status"]
	paymentID := formValues["payment_id"]
	referenceNumber := formValues["reference_number"]

	log.Printf("[ClientHitPayWebhook] schema=%s status=%s payment_id=%s reference=%s", schema, status, paymentID, referenceNumber)

	switch status {
	case "completed":
		return serv.markClientPaid(schema, paymentID)
	case "failed":
		return serv.markClientFailed(schema, paymentID)
	}

	return nil
}

func (serv *PaymentServImpl) markClientPaid(schema, invoiceID string) error {
	orderPayment, err := serv.OrderPaymentRepo.GetByPaymentInvoiceID(serv.Db, schema, invoiceID)
	if err != nil {
		log.Printf("[markClientPaid] order payment not found: %v", err)
		return fmt.Errorf("order payment not found")
	}

	if err := serv.OrderPaymentRepo.UpdateStatus(serv.Db, schema, orderPayment.ID, domains.PaymentStatusPaid); err != nil {
		log.Printf("[markClientPaid] UpdateStatus error: %v", err)
		return fmt.Errorf("failed to update payment status")
	}

	order, err := serv.OrderRepo.GetByID(serv.Db, schema, orderPayment.OrderID)
	if err != nil {
		log.Printf("[markClientPaid] order not found: %v", err)
		return fmt.Errorf("order not found")
	}

	if err := serv.OrderRepo.UpdateStatus(serv.Db, schema, order.ID, domains.OrderStatusConfirmed); err != nil {
		log.Printf("[markClientPaid] UpdateStatus order error: %v", err)
		return fmt.Errorf("failed to update order status")
	}

	log.Printf("[markClientPaid] order #%d confirmed", order.ID)
	return nil
}

func (serv *PaymentServImpl) markClientFailed(schema, invoiceID string) error {
	orderPayment, err := serv.OrderPaymentRepo.GetByPaymentInvoiceID(serv.Db, schema, invoiceID)
	if err != nil {
		log.Printf("[markClientFailed] order payment not found: %v", err)
		return fmt.Errorf("order payment not found")
	}

	if err := serv.OrderPaymentRepo.UpdateStatus(serv.Db, schema, orderPayment.ID, domains.PaymentStatusUnpaid); err != nil {
		log.Printf("[markClientFailed] UpdateStatus error: %v", err)
		return fmt.Errorf("failed to update payment status")
	}

	log.Printf("[markClientFailed] order #%d marked unpaid", orderPayment.OrderID)
	return nil
}
