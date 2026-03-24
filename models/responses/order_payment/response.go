package order_payment

import (
	"backend/models/domains"
	"time"
)

type OrderPaymentResponse struct {
	ID            string    `json:"id"`
	OrderID       int       `json:"order_id"`
	PaymentStatus string    `json:"payment_status"`
	PaymentMethod string    `json:"payment_method"`
	TotalPrice    float64   `json:"total_price"`
	ExpireAt      time.Time `json:"expire_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func ToOrderPaymentResponse(p domains.OrderPayment) OrderPaymentResponse {
	return OrderPaymentResponse{
		ID:            p.ID.String(),
		OrderID:       p.OrderID,
		PaymentStatus: string(p.PaymentStatus),
		PaymentMethod: p.PaymentMethod,
		TotalPrice:    p.TotalPrice,
		ExpireAt:      p.ExpireAt,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func ToOrderPaymentResponses(payments []domains.OrderPayment) []OrderPaymentResponse {
	var responses []OrderPaymentResponse
	for _, p := range payments {
		responses = append(responses, ToOrderPaymentResponse(p))
	}
	return responses
}
