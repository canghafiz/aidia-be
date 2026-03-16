package user

import (
	"backend/models/domains"
)

type UpdateProfileRequest struct {
	Username     string `json:"username"      validate:"omitempty,min=3,max=150"`
	Name         string `json:"name"          validate:"omitempty,min=1,max=150"`
	Email        string `json:"email"         validate:"omitempty,email"`
	Gender       string `json:"gender"        validate:"omitempty,oneof=Male Female"`
	BusinessName string `json:"business_name" validate:"omitempty,max=150"`
	Phone        string `json:"phone"         validate:"omitempty,max=20"`
	Address      string `json:"address"       validate:"omitempty,max=255"`
}

func UpdateProfileRequestToDomain(request UpdateProfileRequest) domains.Users {
	user := domains.Users{
		Username: request.Username,
		Name:     request.Name,
		Email:    request.Email,
		Gender:   request.Gender,
	}

	if request.BusinessName != "" || request.Phone != "" || request.Address != "" {
		user.Tenant = &domains.Tenant{
			BusinessProfile: &domains.BusinessProfile{
				BusinessName: request.BusinessName,
				Phone:        request.Phone,
				Address:      request.Address,
			},
		}
	}

	return user
}
