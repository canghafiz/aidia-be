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
	Db             *gorm.DB
	JwtKey         string
	TenantPlanRepo repositories.TenantPlanRepo
	PlanRepo       repositories.PlanRepo
	TenantRepo     repositories.TenantRepo
	SettingRepo    repositories.SettingRepo
}

func NewPaymentServImpl(
	db *gorm.DB,
	jwtKey string,
	tenantPlanRepo repositories.TenantPlanRepo,
	planRepo repositories.PlanRepo,
	tenantRepo repositories.TenantRepo,
	settingRepo repositories.SettingRepo,
) *PaymentServImpl {
	return &PaymentServImpl{
		Db:             db,
		JwtKey:         jwtKey,
		TenantPlanRepo: tenantPlanRepo,
		PlanRepo:       planRepo,
		TenantRepo:     tenantRepo,
		SettingRepo:    settingRepo,
	}
}

// ============================================================
// HELPER
// ============================================================

func (serv *PaymentServImpl) getStripeKey(schema, subGroupName, name string) (string, error) {
	settings, err := serv.SettingRepo.GetByGroupAndSubGroupName(serv.Db, schema, "integration", subGroupName)
	if err != nil || len(settings) == 0 {
		return "", fmt.Errorf("integration config not found for sub group: %s", subGroupName)
	}
	for _, s := range settings {
		if s.Name == name {
			return s.Value, nil
		}
	}
	return "", fmt.Errorf("stripe key not found: %s", name)
}

// buildInvoice membuat Stripe Customer + InvoiceItem + Invoice
// Return: invoiceID, hostedInvoiceURL, error
func (serv *PaymentServImpl) buildInvoice(
	tenantName string,
	tenantEmail string,
	invoiceNumber string,
	planName string,
	duration string,
	priceInCents int64,
	tenantPlanID uuid.UUID,
) (string, string, error) {
	// 1. Buat customer
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

	// 2. Buat invoice dulu (kosong)
	invoiceParams := &stripe.InvoiceParams{
		Customer:         stripe.String(customer.ID),
		CollectionMethod: stripe.String("send_invoice"),
		DaysUntilDue:     stripe.Int64(7),
		Metadata: map[string]string{
			"tenant_plan_id": tenantPlanID.String(),
			"invoice_number": invoiceNumber,
		},
	}
	inv, err := stripeinvoice.New(invoiceParams)
	if err != nil {
		return "", "", fmt.Errorf("failed to create invoice: %w", err)
	}

	// 3. Buat invoice item dan attach ke invoice
	itemParams := &stripe.InvoiceItemParams{
		Customer:    stripe.String(customer.ID),
		Invoice:     stripe.String(inv.ID),
		Amount:      stripe.Int64(priceInCents),
		Currency:    stripe.String("sgd"),
		Description: stripe.String(fmt.Sprintf("%s | Invoice: %s | Duration: %s", planName, invoiceNumber, duration)),
	}
	_, err = stripeinvoiceitem.New(itemParams)
	if err != nil {
		return "", "", fmt.Errorf("failed to create invoice item: %w", err)
	}
	log.Printf("[buildInvoice] item attached, priceInCents: %d", priceInCents)

	// 4. Finalize invoice
	finalInv, err := stripeinvoice.FinalizeInvoice(inv.ID, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to finalize invoice: %w", err)
	}
	log.Printf("[buildInvoice] invoice finalized: %s, amount_due: %d, hosted_url: %s", finalInv.ID, finalInv.AmountDue, finalInv.HostedInvoiceURL)

	return finalInv.ID, finalInv.HostedInvoiceURL, nil
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

// ============================================================
// PLATFORM - Stripe Aidia
// ============================================================

func (serv *PaymentServImpl) CreatePlatformCheckout(accessToken string, planID uuid.UUID) (*paymentRes.CheckoutResponse, error) {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"Client"})
	if err != nil || !ok {
		return nil, err
	}

	tenant, err := serv.getTenantByToken(accessToken)
	if err != nil {
		return nil, err
	}

	secretKey, err := serv.getStripeKey("public", "Stripe Aidia", "stripe-aidia-secret-key")
	if err != nil {
		return nil, err
	}
	stripe.Key = secretKey

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
	dueDate := time.Now().Add(7 * 24 * time.Hour)

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

	// Ambil nama & email tenant
	tenantName := tenant.User.Name
	tenantEmail := tenant.User.Email
	priceInCents := int64(plan.Price * 100)
	duration := paymentRes.FormatDuration(plan.Duration, plan.IsMonth)

	stripeInvoiceID, hostedURL, err := serv.buildInvoice(
		tenantName, tenantEmail, invoiceNumber, plan.Name, duration, priceInCents, tenantPlan.ID,
	)
	if err != nil {
		tx.Rollback()
		log.Printf("[Stripe].buildInvoice error: %v", err)
		return nil, fmt.Errorf("failed to create payment invoice")
	}

	pending := "pending"
	if err := serv.TenantPlanRepo.UpdateStripeSession(tx, domains.TenantPlan{
		ID:                          tenantPlan.ID,
		StripeSessionID:             &stripeInvoiceID,
		StripeSessionURL:            &hostedURL,
		StripeSubscriptionInvoiceID: &stripeInvoiceID,
		StripePaymentStatus:         &pending,
	}); err != nil {
		tx.Rollback()
		log.Printf("[TenantPlanRepo].UpdateStripeSession error: %v", err)
		return nil, fmt.Errorf("failed to update session")
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to commit transaction")
	}

	return &paymentRes.CheckoutResponse{
		InvoiceID:  tenantPlan.ID,
		SessionID:  stripeInvoiceID,
		SessionURL: hostedURL,
	}, nil
}

func (serv *PaymentServImpl) CreatePaymentFromExisting(accessToken string, tenantPlanID uuid.UUID) (*paymentRes.CheckoutResponse, error) {
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

	secretKey, err := serv.getStripeKey("public", "Stripe Aidia", "stripe-aidia-secret-key")
	if err != nil {
		return nil, err
	}
	stripe.Key = secretKey

	tenantName := tenant.User.Name
	tenantEmail := tenant.User.Email
	priceInCents := int64(tenantPlan.Price * 100)
	duration := paymentRes.FormatDuration(tenantPlan.Duration, tenantPlan.IsMonth)

	stripeInvoiceID, hostedURL, err := serv.buildInvoice(
		tenantName, tenantEmail, tenantPlan.InvoiceNumber, tenantPlan.Plan.Name, duration, priceInCents, tenantPlan.ID,
	)
	if err != nil {
		log.Printf("[Stripe].buildInvoice error: %v", err)
		return nil, fmt.Errorf("failed to create payment invoice")
	}

	pending := "pending"
	if err := serv.TenantPlanRepo.UpdateStripeSession(serv.Db, domains.TenantPlan{
		ID:                          tenantPlan.ID,
		StripeSessionID:             &stripeInvoiceID,
		StripeSessionURL:            &hostedURL,
		StripeSubscriptionInvoiceID: &stripeInvoiceID,
		StripePaymentStatus:         &pending,
	}); err != nil {
		log.Printf("[TenantPlanRepo].UpdateStripeSession error: %v", err)
		return nil, fmt.Errorf("failed to update session")
	}

	return &paymentRes.CheckoutResponse{
		InvoiceID:  tenantPlan.ID,
		SessionID:  stripeInvoiceID,
		SessionURL: hostedURL,
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

func (serv *PaymentServImpl) HandlePlatformWebhook(payload []byte, signature string) error {
	webhookSecret, err := serv.getStripeKey("public", "Stripe Aidia", "stripe-aidia-webhook-secret")
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
		log.Printf("[Webhook] ConstructEvent error detail: %v", errEvent)
		return fmt.Errorf("invalid webhook signature")
	}

	switch event.Type {
	case "invoice.paid":
		invoiceID, ok := event.Data.Object["id"].(string)
		if !ok {
			return fmt.Errorf("invalid invoice data")
		}

		log.Printf("[Webhook] invoice paid, invoiceID: %s", invoiceID)

		tenantPlan, err := serv.TenantPlanRepo.GetByStripeInvoiceID(serv.Db, invoiceID)
		if err != nil {
			log.Printf("[Webhook] GetByStripeInvoiceID error: %v, invoiceID: %s", err, invoiceID)
			return fmt.Errorf("tenant plan not found")
		}

		// Idempotent — skip kalau sudah paid
		if tenantPlan.IsPaid {
			return nil
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
			ID:                  tenantPlan.ID,
			IsPaid:              true,
			PlanStatus:          "Active",
			StripePaymentStatus: &status,
			PaidAt:              &now,
			StartDate:           &now,
			ExpiredDate:         &expiredDate,
		}); err != nil {
			log.Printf("[TenantPlanRepo].UpdatePaymentStatus error: %v", err)
			return fmt.Errorf("failed to update payment status")
		}

	case "invoice.payment_failed":
		invoiceID, ok := event.Data.Object["id"].(string)
		if !ok {
			return fmt.Errorf("invalid invoice data")
		}

		tenantPlan, err := serv.TenantPlanRepo.GetByStripeInvoiceID(serv.Db, invoiceID)
		if err != nil {
			return fmt.Errorf("tenant plan not found")
		}

		if tenantPlan.IsPaid {
			return nil
		}

		status := "failed"
		msg := "Payment failed"
		if err := serv.TenantPlanRepo.UpdatePaymentStatus(serv.Db, domains.TenantPlan{
			ID:                   tenantPlan.ID,
			IsPaid:               false,
			PlanStatus:           "Inactive",
			StripePaymentStatus:  &status,
			StripePaymentMessage: &msg,
		}); err != nil {
			log.Printf("[TenantPlanRepo].UpdatePaymentStatus error: %v", err)
		}
	}

	return nil
}

// ============================================================
// CLIENT - Stripe per tenant
// ============================================================

func (serv *PaymentServImpl) CreateClientCheckout(accessToken string, orderID uuid.UUID) (*paymentRes.CheckoutResponse, error) {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"Client"})
	if err != nil || !ok {
		return nil, err
	}

	schema, err := helpers.GetUsernameFromToken(accessToken, serv.JwtKey)
	if err != nil {
		return nil, err
	}

	secretKey, err := serv.getStripeKey(*schema, "Stripe Client", "stripe-client-secret-key")
	if err != nil {
		return nil, err
	}
	stripe.Key = secretKey

	successURL := os.Getenv("CLIENT_PAYMENT_SUCCESS_URL")
	cancelURL := os.Getenv("CLIENT_PAYMENT_CANCEL_URL")
	if successURL == "" {
		return nil, fmt.Errorf("CLIENT_PAYMENT_SUCCESS_URL env not set")
	}
	if cancelURL == "" {
		return nil, fmt.Errorf("CLIENT_PAYMENT_CANCEL_URL env not set")
	}

	// Placeholder — sesuaikan dengan order domain tenant
	return &paymentRes.CheckoutResponse{
		InvoiceID:  orderID,
		SessionID:  "",
		SessionURL: "",
	}, nil
}

func (serv *PaymentServImpl) GetClientInvoices(accessToken string, pg domains.Pagination) (*pagination.Response, error) {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"Client"})
	if err != nil || !ok {
		return nil, err
	}

	// Placeholder — sesuaikan dengan order domain tenant
	return &pagination.Response{}, nil
}

func (serv *PaymentServImpl) HandleClientWebhook(schema string, payload []byte, signature string) error {
	webhookSecret, err := serv.getStripeKey(schema, "Stripe Client", "stripe-client-webhook-secret")
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
		log.Printf("[ClientWebhook] ConstructEvent error: %v", errEvent)
		return fmt.Errorf("invalid webhook signature")
	}

	switch event.Type {
	case "invoice.paid":
		log.Printf("[ClientWebhook] Payment success for schema: %s, event: %s", schema, event.ID)
	case "invoice.payment_failed":
		log.Printf("[ClientWebhook] Payment failed for schema: %s, event: %s", schema, event.ID)
	}

	return nil
}
