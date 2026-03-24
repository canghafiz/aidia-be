package services

import (
	"backend/models/domains"
	reqOP "backend/models/requests/order_payment"
	resOP "backend/models/responses/order_payment"
	"backend/models/responses/pagination"

	"github.com/google/uuid"
)

type OrderPaymentServ interface {
	GetAll(accessToken string, clientID uuid.UUID, pg domains.Pagination) (*pagination.Response, error)
	GetByID(accessToken string, clientID uuid.UUID, id uuid.UUID) (*resOP.OrderPaymentResponse, error)
	UpdateStatus(accessToken string, clientID uuid.UUID, id uuid.UUID, request reqOP.UpdatePaymentStatusRequest) error
}
