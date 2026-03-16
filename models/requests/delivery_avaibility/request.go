package delivery_avaibility

import (
	"backend/models/domains"

	"github.com/google/uuid"
)

type CreateDeliveryAvailabilityRequest struct {
	Type             string `json:"type"              validate:"required,oneof=available unavailable"`
	DeliverySubGroup string `json:"delivery_subgroup" validate:"required"`
	DateRange        string `json:"date_range"        validate:"required"`
}

type UpdateDeliveryAvailabilityRequest struct {
	Type             string `json:"type"              validate:"required,oneof=available unavailable"`
	DeliverySubGroup string `json:"delivery_subgroup" validate:"required"`
	DateRange        string `json:"date_range"        validate:"required"`
}

func CreateDeliveryAvailabilityToDomain(req CreateDeliveryAvailabilityRequest) []domains.Setting {
	subGroupName := uuid.New().String()
	return []domains.Setting{
		{GroupName: "delivery-availability", SubgroupName: subGroupName, Name: "type", Value: req.Type},
		{GroupName: "delivery-availability", SubgroupName: subGroupName, Name: "delivery_subgroup", Value: req.DeliverySubGroup},
		{GroupName: "delivery-availability", SubgroupName: subGroupName, Name: "date-range", Value: req.DateRange},
	}
}

func UpdateDeliveryAvailabilityToDomain(req UpdateDeliveryAvailabilityRequest, subGroupName string) []domains.Setting {
	return []domains.Setting{
		{GroupName: "delivery-availability", SubgroupName: subGroupName, Name: "type", Value: req.Type},
		{GroupName: "delivery-availability", SubgroupName: subGroupName, Name: "delivery_subgroup", Value: req.DeliverySubGroup},
		{GroupName: "delivery-availability", SubgroupName: subGroupName, Name: "date-range", Value: req.DateRange},
	}
}
