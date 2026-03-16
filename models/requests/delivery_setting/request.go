package delivery

import (
	"backend/models/domains"

	"github.com/google/uuid"
)

// ============================================================
// REQUEST
// ============================================================

type CreateDeliverySettingRequest struct {
	Name        string `json:"name"        validate:"required,max=100"`
	IsVisible   bool   `json:"is_visible"`
	Description string `json:"description" validate:"omitempty,max=255"`
}

type UpdateDeliverySettingRequest struct {
	Name        string `json:"name"        validate:"required,max=100"`
	IsVisible   bool   `json:"is_visible"`
	Description string `json:"description" validate:"omitempty,max=255"`
}

func CreateDeliverySettingToDomain(req CreateDeliverySettingRequest) []domains.Setting {
	subGroupName := uuid.New().String()
	isVisible := "false"
	if req.IsVisible {
		isVisible = "true"
	}
	return []domains.Setting{
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "name", Value: req.Name},
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "is_visible", Value: isVisible},
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "description", Value: req.Description},
	}
}

func UpdateDeliverySettingToDomain(req UpdateDeliverySettingRequest, subGroupName string) []domains.Setting {
	isVisible := "false"
	if req.IsVisible {
		isVisible = "true"
	}
	return []domains.Setting{
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "name", Value: req.Name},
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "is_visible", Value: isVisible},
		{GroupName: "delivery", SubgroupName: subGroupName, Name: "description", Value: req.Description},
	}
}
