package services

import (
	"backend/models/domains"
	"backend/models/responses/pagination"

	"github.com/google/uuid"
)

type ApprovalServ interface {
	Approve(accessToken string, approveId uuid.UUID) error
	GetAll(accessToken string, pagination domains.Pagination) (*pagination.Response, error)
	Delete(accessToken string, approveId uuid.UUID) error
}
