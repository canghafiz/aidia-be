package customer

import "backend/models/domains"

type CreateCustomerRequest struct {
	Name             string `json:"name"                validate:"required,max=150"`
	PhoneCountryCode string `json:"phone_country_code"  validate:"required,max=5"`
	PhoneNumber      string `json:"phone_number"        validate:"required,max=20"`
	AccountType      string `json:"account_type"        validate:"required,max=20"`
}

func CreateCustomerToDomain(req CreateCustomerRequest) domains.Customer {
	return domains.Customer{
		Name:             req.Name,
		PhoneCountryCode: req.PhoneCountryCode,
		PhoneNumber:      req.PhoneNumber,
		AccountType:      req.AccountType,
	}
}
