package delivery_avaibility

import "backend/models/domains"

type DeliveryAvailabilityResponse struct {
	SubGroupName     string `json:"sub_group_name"`
	Type             string `json:"type"`
	DeliverySubGroup string `json:"delivery_subgroup"`
	DeliveryName     string `json:"delivery_name"`
	DateRange        string `json:"date_range"`
}

func ToDeliveryAvailabilityResponse(d domains.DeliveryAvailabilitySetting, deliveryName string) DeliveryAvailabilityResponse {
	return DeliveryAvailabilityResponse{
		SubGroupName:     d.SubGroupName,
		Type:             d.Type,
		DeliverySubGroup: d.DeliverySubGroup,
		DeliveryName:     deliveryName,
		DateRange:        d.DateRange,
	}
}

func ToDeliveryAvailabilityResponses(ds []domains.DeliveryAvailabilitySetting, deliveryMap map[string]string) []DeliveryAvailabilityResponse {
	var responses []DeliveryAvailabilityResponse
	for _, d := range ds {
		deliveryName := deliveryMap[d.DeliverySubGroup]
		responses = append(responses, ToDeliveryAvailabilityResponse(d, deliveryName))
	}
	return responses
}
