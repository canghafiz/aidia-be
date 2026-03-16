package user

import "backend/models/domains"

type BusinessProfileResponse struct {
	ID           string `json:"id"`
	BusinessName string `json:"business_name"`
	Address      string `json:"address"`
	Phone        string `json:"phone"`
}

type TenantResponse struct {
	TenantID        string                   `json:"tenant_id"`
	Role            string                   `json:"role"`
	IsActive        bool                     `json:"is_active"`
	BusinessProfile *BusinessProfileResponse `json:"business_profile,omitempty"`
}

type Response struct {
	UserID   string          `json:"user_id"`
	Username string          `json:"username"`
	Name     string          `json:"name"`
	Email    string          `json:"email"`
	Gender   string          `json:"gender"`
	Role     string          `json:"role"`
	Tenant   *TenantResponse `json:"tenant,omitempty"`
}

func ToResponse(user domains.Users, role string) Response {
	resp := Response{
		UserID:   user.UserID.String(),
		Username: user.Username,
		Name:     user.Name,
		Email:    user.Email,
		Gender:   user.Gender,
		Role:     role,
	}

	if user.Tenant != nil {
		tenantResp := &TenantResponse{
			TenantID: user.Tenant.TenantID.String(),
			Role:     user.Tenant.Role,
			IsActive: user.Tenant.IsActive,
		}

		if user.Tenant.BusinessProfile != nil {
			tenantResp.BusinessProfile = &BusinessProfileResponse{
				ID:           user.Tenant.BusinessProfile.ID.String(),
				BusinessName: user.Tenant.BusinessProfile.BusinessName,
				Address:      user.Tenant.BusinessProfile.Address,
				Phone:        user.Tenant.BusinessProfile.Phone,
			}
		}

		resp.Tenant = tenantResp
	}

	return resp
}
