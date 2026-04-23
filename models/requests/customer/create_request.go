package customer

import "backend/models/domains"

type CreateTelegramCustomerRequest struct {
	Name     string `json:"name"     validate:"required,max=150"`
	Username string `json:"username" validate:"required,max=100"`
}

type CreateCustomerRequest struct {
	AccountType      string `json:"account_type"       validate:"required,max=50"`
	Name             string `json:"name"               validate:"required,max=150"`
	Username         string `json:"username,omitempty" validate:"max=100"`
	PhoneCountryCode string `json:"phone_country_code" validate:"max=5"`
	PhoneNumber      string `json:"phone_number"       validate:"max=20"`
}

func CreateTelegramCustomerToDomain(req CreateTelegramCustomerRequest) domains.Customer {
	return domains.Customer{
		Name:        req.Name,
		Username:    &req.Username,
		AccountType: "Telegram",
	}
}

type CreateWhatsAppCustomerRequest struct {
	Name             string `json:"name"               validate:"required,max=150"`
	PhoneCountryCode string `json:"phone_country_code" validate:"required,max=5"`
	PhoneNumber      string `json:"phone_number"       validate:"required,max=20"`
}

func CreateWhatsAppCustomerToDomain(req CreateWhatsAppCustomerRequest) domains.Customer {
	return domains.Customer{
		Name:             req.Name,
		PhoneCountryCode: &req.PhoneCountryCode,
		PhoneNumber:      &req.PhoneNumber,
		AccountType:      "Whatsapp",
	}
}
