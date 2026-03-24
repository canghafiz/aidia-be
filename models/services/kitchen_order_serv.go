package services

import (
	reqKitchen "backend/models/requests/kitchen_order"
	resKitchen "backend/models/responses/kitchen_order"

	"github.com/google/uuid"
)

type KitchenOrderServ interface {
	GetDisplay(accessToken string, clientID uuid.UUID) (*resKitchen.KitchenDisplayResponse, error)
	UpdateStatus(accessToken string, clientID uuid.UUID, id uuid.UUID, request reqKitchen.UpdateKitchenStatusRequest) error
}
