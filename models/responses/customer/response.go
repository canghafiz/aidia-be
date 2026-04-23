package customer

import (
	"backend/models/domains"
	"time"
)

type Response struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Username         *string   `json:"username,omitempty"`
	PhoneCountryCode *string   `json:"phone_country_code,omitempty"`
	PhoneNumber      *string   `json:"phone_number,omitempty"`
	GuestID          *string   `json:"guest_id,omitempty"`
	AccountType      string    `json:"account_type"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func ToResponse(c domains.Customer) Response {
	return Response{
		ID:               c.ID,
		Name:             c.Name,
		Username:         c.Username,
		PhoneCountryCode: c.PhoneCountryCode,
		PhoneNumber:      c.PhoneNumber,
		AccountType:      c.AccountType,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}
}

func ToResponses(customers []domains.Customer) []Response {
	var responses []Response
	for _, c := range customers {
		responses = append(responses, ToResponse(c))
	}
	return responses
}
