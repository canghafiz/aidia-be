package order

import "backend/models/domains"

type UpdateOrderStatusRequest struct {
	Status domains.OrderStatus `json:"status" validate:"required,oneof=Pending Confirmed Completed Cancelled"`
}
