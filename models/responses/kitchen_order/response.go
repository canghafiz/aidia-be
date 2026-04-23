package kitchen_order

import (
	"backend/models/domains"
	"time"
)

// ============================================================
// RESPONSE
// ============================================================

type CustomerResponse struct {
	ID               int     `json:"id"`
	Name             string  `json:"name"`
	PhoneCountryCode *string `json:"phone_country_code,omitempty"`
	PhoneNumber      *string `json:"phone_number,omitempty"`
}

type OrderProductResponse struct {
	ID         int     `json:"id"`
	ProductID  string  `json:"product_id"`
	Quantity   int     `json:"quantity"`
	TotalPrice float64 `json:"total_price"`
}

type KitchenOrderResponse struct {
	ID            string                 `json:"id"`
	OrderID       int                    `json:"order_id"`
	Status        string                 `json:"status"`
	Customer      *CustomerResponse      `json:"customer"`
	TotalPrice    float64                `json:"total_price"`
	StreetAddress string                 `json:"street_address"`
	Products      []OrderProductResponse `json:"products"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type KitchenDisplayResponse struct {
	NewOrder []KitchenOrderResponse `json:"new_order"`
	Cooking  []KitchenOrderResponse `json:"cooking"`
	Packing  []KitchenOrderResponse `json:"packing"`
	Ready    []KitchenOrderResponse `json:"ready"`
}

type KitchenSSEEvent struct {
	Type string      `json:"type"` // "init", "update"
	Data interface{} `json:"data"`
}

// ============================================================
// MAPPER
// ============================================================

func ToKitchenOrderResponse(ko domains.KitchenOrder) KitchenOrderResponse {
	res := KitchenOrderResponse{
		ID:        ko.ID.String(),
		OrderID:   ko.OrderID,
		Status:    string(ko.Status),
		CreatedAt: ko.CreatedAt,
		UpdatedAt: ko.UpdatedAt,
	}

	if ko.Order != nil {
		res.TotalPrice = ko.Order.TotalPrice
		res.StreetAddress = ko.Order.StreetAddress

		if ko.Order.Customer != nil {
			res.Customer = &CustomerResponse{
				ID:               ko.Order.Customer.ID,
				Name:             ko.Order.Customer.Name,
				PhoneCountryCode: ko.Order.Customer.PhoneCountryCode,
				PhoneNumber:      ko.Order.Customer.PhoneNumber,
			}
		}

		var products []OrderProductResponse
		for _, p := range ko.Order.Products {
			products = append(products, OrderProductResponse{
				ID:         p.ID,
				ProductID:  p.ProductID,
				Quantity:   p.Quantity,
				TotalPrice: p.TotalPrice,
			})
		}
		res.Products = products
	}

	return res
}

func ToKitchenDisplayResponse(orders []domains.KitchenOrder) KitchenDisplayResponse {
	display := KitchenDisplayResponse{
		NewOrder: []KitchenOrderResponse{},
		Cooking:  []KitchenOrderResponse{},
		Packing:  []KitchenOrderResponse{},
		Ready:    []KitchenOrderResponse{},
	}

	for _, ko := range orders {
		res := ToKitchenOrderResponse(ko)
		switch ko.Status {
		case domains.KitchenStatusNewOrder:
			display.NewOrder = append(display.NewOrder, res)
		case domains.KitchenStatusCooking:
			display.Cooking = append(display.Cooking, res)
		case domains.KitchenStatusPacking:
			display.Packing = append(display.Packing, res)
		case domains.KitchenStatusReady:
			display.Ready = append(display.Ready, res)
		}
	}

	return display
}
