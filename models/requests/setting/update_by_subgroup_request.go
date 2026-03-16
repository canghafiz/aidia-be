package setting

import "backend/models/domains"

type UpdateBySubgroupRequest struct {
	Settings []UpdateSettingItem `json:"settings" validate:"required,dive"`
}

type UpdateSettingItem struct {
	Name  string `json:"name" validate:"required"`
	Value string `json:"value" validate:"required"`
}

func UpdateSettingItemToSetting(subGroupName string, request UpdateSettingItem) domains.Setting {
	return domains.Setting{
		SubgroupName: subGroupName,
		Name:         request.Name,
		Value:        request.Value,
	}
}

func UpdateSettingItemsToSettings(subGroupName string, requests []UpdateSettingItem) []domains.Setting {
	var settings []domains.Setting
	for _, item := range requests {
		settings = append(settings, UpdateSettingItemToSetting(subGroupName, item))
	}
	return settings
}
