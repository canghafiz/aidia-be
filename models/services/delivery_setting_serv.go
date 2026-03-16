package services

import (
	reqDelivery "backend/models/requests/delivery_setting"
	resDelivery "backend/models/responses/delivery_setting"

	"github.com/google/uuid"
)

type DeliverySettingServ interface {
	Create(userId uuid.UUID, request reqDelivery.CreateDeliverySettingRequest) error
	Update(userId uuid.UUID, subGroupName string, request reqDelivery.UpdateDeliverySettingRequest) error
	GetAll(userId uuid.UUID) ([]resDelivery.DeliverySettingResponse, error)
	GetBySubGroupName(userId uuid.UUID, subGroupName string) (*resDelivery.DeliverySettingResponse, error)
	Delete(userId uuid.UUID, subGroupName string) error
}
