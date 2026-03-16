package user

import (
	"backend/models/domains"
)

type UpdateProfileNonClientRequest struct {
	Username string `json:"username"      validate:"omitempty,min=3,max=150"`
	Name     string `json:"name"          validate:"omitempty,min=1,max=150"`
	Email    string `json:"email"         validate:"omitempty,email"`
	Gender   string `json:"gender"        validate:"omitempty,oneof=Male Female"`
}

func UpdateProfileNonClientRequestToDomain(request UpdateProfileNonClientRequest) domains.Users {
	user := domains.Users{
		Username: request.Username,
		Name:     request.Name,
		Email:    request.Email,
		Gender:   request.Gender,
	}

	return user
}
