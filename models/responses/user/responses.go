package user

import "backend/models/domains"

type Responses struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	BusinessName string `json:"business_name"`
	Role         string `json:"role"`
	IsActive     bool   `json:"is_active"`
}

func ToResponses(user domains.Users, role string) Responses {
	resp := Responses{
		UserID:   user.UserID.String(),
		Username: user.Username,
		Name:     user.Name,
		Email:    user.Email,
		Role:     role,
		IsActive: user.IsActive,
	}

	if user.Tenant != nil && user.Tenant.BusinessProfile != nil {
		resp.BusinessName = user.Tenant.BusinessProfile.BusinessName
	}

	return resp
}

func ToResponsesList(users []domains.Users) []Responses {
	var result []Responses
	for _, user := range users {
		role := ""
		if len(user.UserRoles) > 0 {
			role = user.UserRoles[0].Role.Name
		}
		result = append(result, ToResponses(user, role))
	}
	return result
}
