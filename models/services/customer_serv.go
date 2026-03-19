package services

import (
	"backend/models/domains"
	req "backend/models/requests/customer"
	res "backend/models/responses/customer"
	"backend/models/responses/pagination"

	"github.com/google/uuid"
)

type CustomerServ interface {
	Create(accessToken string, clientID uuid.UUID, request req.CreateCustomerRequest) (*res.Response, error)
	GetAll(accessToken string, clientID uuid.UUID, pg domains.Pagination) (*pagination.Response, error)
	GetByID(accessToken string, clientID uuid.UUID, id int) (*res.Response, error)
}
