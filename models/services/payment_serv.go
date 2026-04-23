package services

import (
	"backend/models/domains"
	"backend/models/responses/pagination"
	paymentRes "backend/models/responses/payment"

	"github.com/google/uuid"
)

type PaymentServ interface {
	// Platform — tenant purchases a plan. gateway = "stripe" | "hitpay" (empty → active default)
	CreatePlatformCheckout(accessToken string, planID uuid.UUID, gateway string) (*paymentRes.CheckoutResponse, error)
	CreatePaymentFromExisting(accessToken string, tenantPlanID uuid.UUID, gateway string) (*paymentRes.CheckoutResponse, error)
	GetPlatformInvoices(accessToken string, pg domains.Pagination) (*pagination.Response, error)
	GetPlatformInvoiceByID(accessToken string, id uuid.UUID) (*paymentRes.InvoiceResponse, error)
	GetAvailableGateways() []string

	// Platform webhooks — one endpoint per gateway (signature schemes differ)
	HandlePlatformWebhookStripe(payload []byte, signature string) error
	HandlePlatformWebhookHitPay(formValues map[string]string) error

	// Client — tenant receives payments from their customers
	CreateClientCheckout(clientID uuid.UUID, orderID uuid.UUID) (*paymentRes.CheckoutResponse, error)
	GetClientInvoices(clientID uuid.UUID, pg domains.Pagination) (*pagination.Response, error)
	HandleClientWebhookStripe(schema string, payload []byte, signature string) error
	HandleClientWebhookHitPay(schema string, formValues map[string]string) error
}
