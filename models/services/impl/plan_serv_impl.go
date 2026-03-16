package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	"backend/models/requests/plan"
	"backend/models/responses/pagination"
	res "backend/models/responses/plan"
	"fmt"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlanServImpl struct {
	Db        *gorm.DB
	Validator *validator.Validate
	PlanRepo  repositories.PlanRepo
	JwtKey    string
}

func NewPlanServImpl(db *gorm.DB, validator *validator.Validate, planRepo repositories.PlanRepo, jwtKey string) *PlanServImpl {
	return &PlanServImpl{Db: db, Validator: validator, PlanRepo: planRepo, JwtKey: jwtKey}
}

func (serv *PlanServImpl) Create(accessToken string, request plan.CreateRequest) error {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin"})
	if err != nil || !ok {
		return err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return err
	}

	model := plan.CreateRequestToDomain(request)

	if err := serv.PlanRepo.Create(serv.Db, model); err != nil {
		log.Printf("[PlanServ.Create] error: %v", err)
		return fmt.Errorf("failed to create plan")
	}

	return nil
}

func (serv *PlanServImpl) Update(accessToken string, id uuid.UUID, request plan.UpdateRequest) error {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin"})
	if err != nil || !ok {
		return err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return err
	}

	existing, err := serv.PlanRepo.GetById(serv.Db, id)
	if err != nil || existing == nil {
		return fmt.Errorf("plan not found")
	}

	model := plan.UpdateRequestToDomain(request)
	model.ID = id

	if err := serv.PlanRepo.Update(serv.Db, model); err != nil {
		log.Printf("[PlanServ.Update] error: %v", err)
		return fmt.Errorf("failed to update plan")
	}

	return nil
}

func (serv *PlanServImpl) ToggleIsActive(accessToken string, planId uuid.UUID) error {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin"})
	if err != nil || !ok {
		return err
	}

	existing, err := serv.PlanRepo.GetById(serv.Db, planId)
	if err != nil || existing == nil {
		return fmt.Errorf("plan not found")
	}

	if err := serv.PlanRepo.ToggleIsActive(serv.Db, planId, !existing.IsActive); err != nil {
		log.Printf("[PlanServ.ToggleIsActive] error: %v", err)
		return fmt.Errorf("failed to toggle plan status")
	}

	return nil
}

func (serv *PlanServImpl) GetAll(accessToken string, pg domains.Pagination) (pagination.Response, error) {
	role, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Client"})
	if err != nil || !ok {
		return pagination.Response{}, err
	}

	if *role == "Client" {
		plans, total, err := serv.PlanRepo.GetAllPublic(serv.Db)
		if err != nil {
			log.Printf("[PlanServ.GetAll/Client] error: %v", err)
			return pagination.Response{}, fmt.Errorf("failed to get plans")
		}
		return pagination.ToResponse(
			res.ToPublicResponses(plans),
			total,
			pg.Page,
			pg.Limit,
		), nil
	}

	// SuperAdmin
	plans, total, err := serv.PlanRepo.GetAll(serv.Db, pg)
	if err != nil {
		log.Printf("[PlanServ.GetAll/SuperAdmin] error: %v", err)
		return pagination.Response{}, fmt.Errorf("failed to get plans")
	}

	return pagination.ToResponse(
		res.ToResponses(plans),
		total,
		pg.Page,
		pg.Limit,
	), nil
}

func (serv *PlanServImpl) GetById(accessToken string, id uuid.UUID) (interface{}, error) {
	role, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Client"})
	if err != nil || !ok {
		return nil, err
	}

	if *role == "Client" {
		found, err := serv.PlanRepo.GetByIdPublic(serv.Db, id)
		if err != nil || found == nil {
			return nil, fmt.Errorf("plan not found or not active")
		}
		result := res.ToPublicResponse(*found)
		return &result, nil
	}

	// SuperAdmin
	found, err := serv.PlanRepo.GetById(serv.Db, id)
	if err != nil || found == nil {
		return nil, fmt.Errorf("plan not found")
	}

	result := res.ToResponse(*found)
	return &result, nil
}

func (serv *PlanServImpl) Delete(accessToken string, id uuid.UUID) error {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin"})
	if err != nil || !ok {
		return err
	}

	existing, errGet := serv.PlanRepo.GetById(serv.Db, id)
	if errGet != nil || existing == nil {
		return fmt.Errorf("plan not found")
	}

	if err := serv.PlanRepo.Delete(serv.Db, *existing); err != nil {
		log.Printf("[PlanServ.Delete] error: %v", err)
		return fmt.Errorf("failed to delete plan")
	}

	return nil
}
