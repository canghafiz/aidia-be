package approve

import (
	"backend/models/domains"

	"github.com/google/uuid"
)

type Response struct {
	Id           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PhoneNumber  string    `json:"phone_number"`
	BusinessName string    `json:"business_name"`
	Status       string    `json:"status"`
}

func ToResponse(log domains.TenantApprovalLogs) Response {
	var name, username, email, phone, businessName, status string

	if log.User.Name != "" {
		name = log.User.Name
	}
	if log.User.Username != "" {
		username = log.User.Username
	}
	if log.User.Email != "" {
		email = log.User.Email
	}
	if log.User.Tenant != nil && log.User.Tenant.BusinessProfile != nil {
		phone = log.User.Tenant.BusinessProfile.Phone
		businessName = log.User.Tenant.BusinessProfile.BusinessName
	}
	if log.Action != nil {
		status = *log.Action
	}

	return Response{
		Id:           log.ID,
		Name:         name,
		Username:     username,
		Email:        email,
		PhoneNumber:  phone,
		BusinessName: businessName,
		Status:       status,
	}
}

func ToResponses(logs []domains.TenantApprovalLogs) []Response {
	responses := make([]Response, 0, len(logs))
	for _, log := range logs {
		responses = append(responses, ToResponse(log))
	}
	return responses
}
