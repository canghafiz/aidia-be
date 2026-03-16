package payment

import (
	"backend/models/domains"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type CheckoutResponse struct {
	InvoiceID  uuid.UUID `json:"invoice_id"`
	SessionID  string    `json:"session_id"`
	SessionURL string    `json:"session_url"`
}

type InvoiceResponse struct {
	ID            uuid.UUID  `json:"id"`
	InvoiceNumber string     `json:"invoice_number"`
	PlanName      string     `json:"plan_name"`
	Duration      string     `json:"duration"`
	Price         float64    `json:"price"`
	IsPaid        bool       `json:"is_paid"`
	PaymentStatus string     `json:"payment_status"`
	ServiceStatus string     `json:"service_status"`
	ServiceStart  *time.Time `json:"service_start"`
	ExpiredDate   *time.Time `json:"service_expired_on"`
	PaidAt        *time.Time `json:"paid_at"`
	SessionURL    *string    `json:"session_url"`
}

func ToInvoiceResponse(tp domains.TenantPlan) InvoiceResponse {
	paymentStatus := "Unpaid"
	if tp.IsPaid {
		paymentStatus = "Paid"
	}

	// ServiceStatus dari plan_status field
	serviceStatus := tp.PlanStatus
	if serviceStatus == "" {
		serviceStatus = "Inactive"
	}

	return InvoiceResponse{
		ID:            tp.ID,
		InvoiceNumber: tp.InvoiceNumber,
		PlanName:      tp.Plan.Name,
		Duration:      FormatDuration(tp.Duration, tp.IsMonth),
		Price:         tp.Price,
		IsPaid:        tp.IsPaid,
		PaymentStatus: paymentStatus,
		ServiceStatus: serviceStatus,
		ServiceStart:  tp.StartDate,
		ExpiredDate:   tp.ExpiredDate,
		PaidAt:        tp.PaidAt,
		SessionURL:    tp.StripeSessionURL,
	}
}

func ToInvoiceResponses(plans []domains.TenantPlan) []InvoiceResponse {
	responses := make([]InvoiceResponse, 0, len(plans))
	for _, p := range plans {
		responses = append(responses, ToInvoiceResponse(p))
	}
	return responses
}

func FormatDuration(duration int, isMonth bool) string {
	if isMonth {
		if duration == 1 {
			return "1 Month"
		}
		return fmt.Sprintf("%d Month", duration)
	}
	if duration == 1 {
		return "1 Year"
	}
	return fmt.Sprintf("%d Year", duration)
}

func StrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
