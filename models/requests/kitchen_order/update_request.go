package kitchen_order

import "backend/models/domains"

type UpdateKitchenStatusRequest struct {
	Status domains.KitchenStatus `json:"status" validate:"required,oneof=new_order cooking packing ready"`
}
