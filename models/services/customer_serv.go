package services

import (
	"backend/models/domains"
	req "backend/models/requests/customer"
	res "backend/models/responses/customer"
	"backend/models/responses/pagination"

	"github.com/google/uuid"
)

type CustomerServ interface {
	CreateTelegram(accessToken string, clientID uuid.UUID, request req.CreateTelegramCustomerRequest) (*res.Response, error)
	CreateWhatsApp(accessToken string, clientID uuid.UUID, request req.CreateWhatsAppCustomerRequest) (*res.Response, error)
	Update(accessToken string, clientID uuid.UUID, customerID int, request req.CreateCustomerRequest) (*res.Response, error)
	GetAll(accessToken string, clientID uuid.UUID, pg domains.Pagination) (*pagination.Response, error)
	GetByID(accessToken string, clientID uuid.UUID, id int) (*res.Response, error)
}
