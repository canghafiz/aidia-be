package order

type CreateOrderProductRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity"   validate:"required,min=1"`
}

type CreateOrderRequest struct {
	PhoneCountryCode     string                      `json:"phone_country_code"      validate:"required,max=5"`
	PhoneNumber          string                      `json:"phone_number"            validate:"required,max=20"`
	Name                 string                      `json:"customer_name"                    validate:"required,max=150"`
	DeliverySubGroupName string                      `json:"delivery_sub_group_name" validate:"required"`
	StreetAddress        string                      `json:"street_address"          validate:"required,max=100"`
	PostalCode           string                      `json:"postal_code"             validate:"required,max=20"`
	Products             []CreateOrderProductRequest `json:"products"                validate:"required,min=1,dive"`
}
