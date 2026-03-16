package services

import (
	"backend/models/domains"
	"backend/models/responses/pagination"
	paymentRes "backend/models/responses/payment"

	"github.com/google/uuid"
)

type PaymentServ interface {
	// CreatePlatformCheckout Platform (Stripe Aidia) - tenant beli plan
	CreatePlatformCheckout(accessToken string, planID uuid.UUID) (*paymentRes.CheckoutResponse, error)
	CreatePaymentFromExisting(accessToken string, tenantPlanID uuid.UUID) (*paymentRes.CheckoutResponse, error)
	GetPlatformInvoices(accessToken string, pg domains.Pagination) (*pagination.Response, error)
	GetPlatformInvoiceByID(accessToken string, id uuid.UUID) (*paymentRes.InvoiceResponse, error)
	HandlePlatformWebhook(payload []byte, signature string) error

	// CreateClientCheckout Client (Stripe per tenant) - tenant terima pembayaran dari customer
	CreateClientCheckout(accessToken string, orderID uuid.UUID) (*paymentRes.CheckoutResponse, error)
	GetClientInvoices(accessToken string, pg domains.Pagination) (*pagination.Response, error)
	HandleClientWebhook(schema string, payload []byte, signature string) error
}
