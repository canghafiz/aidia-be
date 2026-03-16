package user

import "backend/models/domains"

type CreateSuperAdminRequest struct {
	Username string `json:"username" validate:"required,min=3,max=150"`
	Email    string `json:"email"    validate:"required,email"`
	Gender   string `json:"gender"   validate:"required,oneof=Male Female"`
	Password string `json:"password" validate:"required,min=6,max=100"`
}

func CreateSuperAdminRequestToUser(req CreateSuperAdminRequest) domains.Users {
	return domains.Users{
		Username: req.Username,
		Email:    req.Email,
		Gender:   req.Gender,
		Password: req.Password,
		Name:     "Super Admin",
		IsActive: true,
	}
}
