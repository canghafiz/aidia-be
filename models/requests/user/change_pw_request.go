package user

import (
	"backend/models/domains"
)

type ChangePwRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,min=6,max=100"`
	NewPassword     string `json:"new_password"     validate:"required,min=6,max=100"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=NewPassword"`
}

func ChangePwToDomain(request ChangePwRequest) domains.Users {
	return domains.Users{
		Password: request.NewPassword,
	}
}
