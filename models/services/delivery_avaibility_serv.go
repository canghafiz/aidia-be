package services

import (
	reqAvailability "backend/models/requests/delivery_avaibility"
	resAvailability "backend/models/responses/delivery_avaibility"

	"github.com/google/uuid"
)

type DeliveryAvailabilitySettingServ interface {
	Create(userId uuid.UUID, request reqAvailability.CreateDeliveryAvailabilityRequest) error
	Update(userId uuid.UUID, subGroupName string, request reqAvailability.UpdateDeliveryAvailabilityRequest) error
	GetAll(userId uuid.UUID) ([]resAvailability.DeliveryAvailabilityResponse, error)
	GetBySubGroupName(userId uuid.UUID, subGroupName string) (*resAvailability.DeliveryAvailabilityResponse, error)
	Delete(userId uuid.UUID, subGroupName string) error
}
