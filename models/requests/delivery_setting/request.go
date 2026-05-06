package delivery

import (
	"backend/models/domains"
	"fmt"

	"github.com/google/uuid"
)

// ============================================================
// REQUEST
// ============================================================

type CreateDeliverySettingRequest struct {
	Name         string  `json:"name"          validate:"required,max=100"`
	IsVisible    bool    `json:"is_visible"`
	Description  string  `json:"description"   validate:"omitempty,max=255"`
	DeliveryType string  `json:"delivery_type" validate:"omitempty,max=50"`
	Charge       float64 `json:"charge"        validate:"omitempty,min=0"`
}

type UpdateDeliverySettingRequest struct {
	Name         string  `json:"name"          validate:"required,max=100"`
	IsVisible    bool    `json:"is_visible"`
	Description  string  `json:"description"   validate:"omitempty,max=255"`
	DeliveryType string  `json:"delivery_type" validate:"omitempty,max=50"`
	Charge       float64 `json:"charge"        validate:"omitempty,min=0"`
}

func CreateDeliverySettingToDomain(req CreateDeliverySettingRequest) []domains.Setting {
	subGroupName := uuid.New().String()
	isVisible := "false"
	if req.IsVisible {
		isVisible = "true"
	}
	deliveryType := req.DeliveryType
	if deliveryType == "" {
		deliveryType = "Delivery"
	}
	return []domains.Setting{
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "name", Value: req.Name},
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "is_visible", Value: isVisible},
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "description", Value: req.Description},
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "delivery_type", Value: deliveryType},
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "charge", Value: fmt.Sprintf("%.2f", req.Charge)},
	}
}

func UpdateDeliverySettingToDomain(req UpdateDeliverySettingRequest, subGroupName string) []domains.Setting {
	isVisible := "false"
	if req.IsVisible {
		isVisible = "true"
	}
	deliveryType := req.DeliveryType
	if deliveryType == "" {
		deliveryType = "Delivery"
	}
	return []domains.Setting{
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "name", Value: req.Name},
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "is_visible", Value: isVisible},
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "description", Value: req.Description},
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "delivery_type", Value: deliveryType},
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "charge", Value: fmt.Sprintf("%.2f", req.Charge)},
	}
}
