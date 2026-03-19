package services

import (
	"backend/models/domains"
	reqOrder "backend/models/requests/order"
	resOrder "backend/models/responses/order"
	"backend/models/responses/pagination"

	"github.com/google/uuid"
)

type OrderServ interface {
	Create(accessToken string, clientID uuid.UUID, request reqOrder.CreateOrderRequest) (*resOrder.DetailResponse, error)
	GetAll(accessToken string, clientID uuid.UUID, pg domains.Pagination) (*pagination.Response, error)
	GetByID(accessToken string, clientID uuid.UUID, id int) (*resOrder.DetailResponse, error)
	UpdateStatus(accessToken string, clientID uuid.UUID, id int, request reqOrder.UpdateOrderStatusRequest) error
}
