package user

import (
	"backend/models/domains"

	"github.com/google/uuid"
)

type CreateUserRequest struct {
	Username     string `json:"username"      validate:"required,min=3,max=150"`
	Name         string `json:"name"          validate:"required,min=1,max=150"`
	Email        string `json:"email"         validate:"required,email"`
	Gender       string `json:"gender"        validate:"required,oneof=Male Female"`
	Password     string `json:"password"      validate:"required,min=6,max=100"`
	RoleID       string `json:"role_id"       validate:"required,uuid"`
	BusinessName string `json:"business_name" validate:"omitempty,max=150"`
	Phone        string `json:"phone"         validate:"omitempty,max=20"`
	Address      string `json:"address"       validate:"omitempty,max=255"`
}

func CreateUserRequestToDomain(request CreateUserRequest) domains.Users {
	roleUUID, _ := uuid.Parse(request.RoleID)

	return domains.Users{
		Username: request.Username,
		Name:     request.Name,
		Email:    request.Email,
		Gender:   request.Gender,
		Password: request.Password,
		UserRoles: []domains.UserRoles{
			{RoleID: roleUUID},
		},
		Tenant: &domains.Tenant{
			BusinessProfile: &domains.BusinessProfile{
				BusinessName: request.BusinessName,
				Phone:        request.Phone,
				Address:      request.Address,
			},
		},
	}
}
