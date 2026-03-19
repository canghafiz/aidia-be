package order

import "backend/models/domains"

type Response struct {
	ID            int     `json:"id"`
	CustomerName  string  `json:"customer_name"`
	TotalPrice    float64 `json:"total_price"`
	OrderStatus   string  `json:"order_status"`
	PaymentStatus string  `json:"payment_status"`
	TotalProduct  int     `json:"total_product"`
	DeliveryName  string  `json:"delivery_name"`
	StreetAddress string  `json:"street_address"`
}

func ToResponse(o domains.Order, deliveryName string) Response {
	customerName := ""
	if o.Customer != nil {
		customerName = o.Customer.Name
	}

	paymentStatus := ""
	if o.Payment != nil {
		paymentStatus = string(o.Payment.PaymentStatus)
	}

	return Response{
		ID:            o.ID,
		CustomerName:  customerName,
		TotalPrice:    o.TotalPrice,
		OrderStatus:   string(o.Status),
		PaymentStatus: paymentStatus,
		TotalProduct:  len(o.Products),
		DeliveryName:  deliveryName,
		StreetAddress: o.StreetAddress,
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
