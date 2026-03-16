package domains

type DeliveryAvailabilitySetting struct {
	SubGroupName     string
	Type             string // available / unavailable
	DeliverySubGroup string
	DateRange        string
}

func ToDeliveryAvailabilitySetting(settings []Setting) []DeliveryAvailabilitySetting {
	grouped := map[string]*DeliveryAvailabilitySetting{}
	var order []string

	for _, s := range settings {
		if s.GroupName != "delivery-availability" {
			continue
		}

		if _, ok := grouped[s.SubgroupName]; !ok {
			grouped[s.SubgroupName] = &DeliveryAvailabilitySetting{
				SubGroupName: s.SubgroupName,
			}
			order = append(order, s.SubgroupName)
		}

		switch s.Name {
		case "type":
			grouped[s.SubgroupName].Type = s.Value
		case "delivery_subgroup":
			grouped[s.SubgroupName].DeliverySubGroup = s.Value
		case "date-range":
			grouped[s.SubgroupName].DateRange = s.Value
		}
	}

	var result []DeliveryAvailabilitySetting
	for _, key := range order {
		result = append(result, *grouped[key])
	}
	return result
}
