package order

import (
	"backend/models/domains"
	"time"
)

type Response struct {
	ID            int       `json:"id"`
	CustomerName  string    `json:"customer_name"`
	TotalPrice    float64   `json:"total_price"`
	OrderStatus   string    `json:"order_status"`
	PaymentStatus string    `json:"payment_status"`
	PaymentMethod string    `json:"payment_method"`
	PaymentID     string    `json:"payment_id"`
	TotalProduct  int       `json:"total_product"`
	TotalQuantity int       `json:"total_quantity"`
	DeliveryName  string    `json:"delivery_name"`
	StreetAddress string    `json:"street_address"`
	CreatedAt     time.Time `json:"created_at"`
}

func ToResponse(o domains.Order, deliveryName string) Response {
	customerName := ""
	if o.Customer != nil {
		customerName = o.Customer.Name
	}

	paymentStatus := ""
	paymentMethod := ""
	paymentID := ""
	if o.Payment != nil {
		paymentStatus = string(o.Payment.PaymentStatus)
		paymentMethod = o.Payment.PaymentMethod
		paymentID = o.Payment.ID.String()
	}

	totalQty := 0
	for _, op := range o.Products {
		totalQty += op.Quantity
	}

	return Response{
		ID:            o.ID,
		CustomerName:  customerName,
		TotalPrice:    o.TotalPrice,
		OrderStatus:   string(o.Status),
		PaymentStatus: paymentStatus,
		PaymentMethod: paymentMethod,
		PaymentID:     paymentID,
		TotalProduct:  len(o.Products),
		TotalQuantity: totalQty,
		DeliveryName:  deliveryName,
		StreetAddress: o.StreetAddress,
		CreatedAt:     o.CreatedAt,
	}
}

func ToResponses(orders []domains.Order, deliveryMap map[string]string) []Response {
	var responses []Response
	for _, o := range orders {
		deliveryName := deliveryMap[o.DeliverySubGroupName]
		responses = append(responses, ToResponse(o, deliveryName))
	}
	return responses
}
