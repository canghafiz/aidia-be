package domains

import (
	"strconv"

	"github.com/google/uuid"
)

type DeliverySetting struct {
	SubGroupName string
	Name         string
	IsVisible    bool
	Description  string
	DeliveryType string
	Charge       float64
}

func ToDeliverySetting(settings []Setting) []DeliverySetting {
	grouped := map[string]*DeliverySetting{}
	var order []string

	for _, s := range settings {
		if _, ok := grouped[s.SubgroupName]; !ok {
			grouped[s.SubgroupName] = &DeliverySetting{
				SubGroupName: s.SubgroupName,
			}
			order = append(order, s.SubgroupName)
		}

		switch s.Name {
		case "name":
			grouped[s.SubgroupName].Name = s.Value
		case "is_visible":
			grouped[s.SubgroupName].IsVisible = s.Value == "true"
		case "description":
			grouped[s.SubgroupName].Description = s.Value
		case "delivery_type":
			grouped[s.SubgroupName].DeliveryType = s.Value
		case "charge":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				grouped[s.SubgroupName].Charge = v
			}
		}
	}

	var result []DeliverySetting
	for _, key := range order {
		result = append(result, *grouped[key])
	}
	return result
}

func ToDeliverySettingList(settings []Setting) []Setting {
	var result []Setting
	subGroupID := uuid.New().String()
	for _, s := range settings {
		if s.GroupName == "delivery" {
			s.SubgroupName = subGroupID
			result = append(result, s)
		}
	}
	return result
}
