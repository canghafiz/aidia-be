package order_payment

import "backend/models/domains"

type UpdatePaymentStatusRequest struct {
	Status domains.PaymentStatus `json:"status" validate:"required,oneof=Unpaid Confirming_Payment Paid Refunded Voided"`
}
