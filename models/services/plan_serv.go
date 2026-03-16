package services

import (
	"backend/models/domains"
	"backend/models/requests/plan"
	"backend/models/responses/pagination"

	"github.com/google/uuid"
)

type PlanServ interface {
	Create(accessToken string, request plan.CreateRequest) error
	Update(accessToken string, id uuid.UUID, request plan.UpdateRequest) error
	ToggleIsActive(accessToken string, planId uuid.UUID) error
	GetAll(accessToken string, pg domains.Pagination) (pagination.Response, error)
	GetById(accessToken string, id uuid.UUID) (interface{}, error)
	Delete(accessToken string, id uuid.UUID) error
}
