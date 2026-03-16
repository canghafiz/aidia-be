package user

import "backend/models/domains"

type SingleResponse struct {
	UserID       string  `json:"user_id"`
	Username     string  `json:"username"`
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	Gender       string  `json:"gender"`
	Role         string  `json:"role"`
	IsActive     bool    `json:"is_active"`
	TenantSchema *string `json:"tenant_schema,omitempty"`
	BusinessName string  `json:"business_name"`
	Phone        string  `json:"phone"`
	Address      string  `json:"address"`
}

func ToSingleResponse(user domains.Users, role string) SingleResponse {
	resp := SingleResponse{
		UserID:       user.UserID.String(),
		Username:     user.Username,
		Name:         user.Name,
		Email:        user.Email,
		Gender:       user.Gender,
		Role:         role,
		IsActive:     user.IsActive,
		TenantSchema: user.TenantSchema,
	}

	if user.Tenant != nil && user.Tenant.BusinessProfile != nil {
		resp.BusinessName = user.Tenant.BusinessProfile.BusinessName
		resp.Phone = user.Tenant.BusinessProfile.Phone
		resp.Address = user.Tenant.BusinessProfile.Address
	}

	return resp
}
