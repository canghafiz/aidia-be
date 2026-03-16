package setting

import (
	"backend/models/domains"
	"sort"
)

type GroupResponse struct {
	GroupName string             `json:"group_name"`
	SubGroups []SubgroupResponse `json:"sub_groups"`
}

type SubgroupResponse struct {
	SubGroupName string     `json:"sub_group_name"`
	SubgroupData []Response `json:"data"`
}

type Response struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func ToResponse(model domains.Setting) Response {
	return Response{
		Name:  model.Name,
		Value: model.Value,
	}
}

func ToGroupResponse(settings []domains.Setting) *GroupResponse {
	subgroupMap := make(map[string][]Response)

	for _, s := range settings {
		subgroupMap[s.SubgroupName] = append(subgroupMap[s.SubgroupName], ToResponse(s))
	}

	var subgroupList []SubgroupResponse
	for subGroupName, data := range subgroupMap {
		sort.Slice(data, func(i, j int) bool {
			return data[i].Name < data[j].Name
		})
		subgroupList = append(subgroupList, SubgroupResponse{
			SubGroupName: subGroupName,
			SubgroupData: data,
		})
	}

	sort.Slice(subgroupList, func(i, j int) bool {
		return subgroupList[i].SubGroupName < subgroupList[j].SubGroupName
	})

	groupName := ""
	if len(settings) > 0 {
		groupName = settings[0].GroupName
	}

	return &GroupResponse{
		GroupName: groupName,
		SubGroups: subgroupList,
	}
}
