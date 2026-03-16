package delivery_setting

import "backend/models/domains"

type DeliverySettingResponse struct {
	SubGroupName string `json:"sub_group_name"`
	Name         string `json:"name"`
	IsVisible    bool   `json:"is_visible"`
	Description  string `json:"description"`
}

func ToDeliverySettingResponse(d domains.DeliverySetting) DeliverySettingResponse {
	return DeliverySettingResponse{
		SubGroupName: d.SubGroupName,
		Name:         d.Name,
		IsVisible:    d.IsVisible,
		Description:  d.Description,
	}
}

func ToDeliverySettingResponses(ds []domains.DeliverySetting) []DeliverySettingResponse {
	var responses []DeliverySettingResponse
	for _, d := range ds {
		responses = append(responses, ToDeliverySettingResponse(d))
	}
	return responses
}
