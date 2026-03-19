package order

import (
	"backend/models/domains"
	"time"
)

type CustomerDetailResponse struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	PhoneCountryCode string `json:"phone_country_code"`
	PhoneNumber      string `json:"phone_number"`
}

type ProductDetailResponse struct {
	ID          int     `json:"id"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
	TotalPrice  float64 `json:"total_price"`
}

type PaymentDetailResponse struct {
	ID            string  `json:"id"`
	PaymentStatus string  `json:"payment_status"`
	PaymentMethod string  `json:"payment_method"`
	TotalPrice    float64 `json:"total_price"`
}

type DetailResponse struct {
	ID                   int                     `json:"id"`
	Customer             *CustomerDetailResponse `json:"customer"`
	TotalPrice           float64                 `json:"total_price"`
	OrderStatus          string                  `json:"order_status"`
	DeliverySubGroupName string                  `json:"delivery_sub_group_name"`
	DeliveryName         string                  `json:"delivery_name"`
	StreetAddress        string                  `json:"street_address"`
	PostalCode           string                  `json:"postal_code"`
	Products             []ProductDetailResponse `json:"products"`
	Payment              *PaymentDetailResponse  `json:"payment"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            time.Time               `json:"updated_at"`
}

func ToDetailResponse(o domains.Order, deliveryName string, products []ProductDetailResponse) DetailResponse {
	var customer *CustomerDetailResponse
	if o.Customer != nil {
		customer = &CustomerDetailResponse{
			ID:               o.Customer.ID,
			Name:             o.Customer.Name,
			PhoneCountryCode: o.Customer.PhoneCountryCode,
			PhoneNumber:      o.Customer.PhoneNumber,
		}
	}

	var payment *PaymentDetailResponse
	if o.Payment != nil {
		payment = &PaymentDetailResponse{
			ID:            o.Payment.ID.String(),
			PaymentStatus: string(o.Payment.PaymentStatus),
			PaymentMethod: o.Payment.PaymentMethod,
			TotalPrice:    o.Payment.TotalPrice,
		}
	}

	return DetailResponse{
		ID:                   o.ID,
		Customer:             customer,
		TotalPrice:           o.TotalPrice,
		OrderStatus:          string(o.Status),
		DeliverySubGroupName: o.DeliverySubGroupName,
		DeliveryName:         deliveryName,
		StreetAddress:        o.StreetAddress,
		PostalCode:           o.PostalCode,
		Products:             products,
		Payment:              payment,
		CreatedAt:            o.CreatedAt,
		UpdatedAt:            o.UpdatedAt,
	}
}
