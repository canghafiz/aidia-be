package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	reqOP "backend/models/requests/order_payment"
	resOP "backend/models/responses/order_payment"
	"backend/models/responses/pagination"
	"fmt"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderPaymentServImpl struct {
	Db               *gorm.DB
	JwtKey           string
	Validator        *validator.Validate
	UserRepo         repositories.UsersRepo
	OrderPaymentRepo repositories.OrderPaymentRepo
}

func NewOrderPaymentServImpl(
	db *gorm.DB,
	jwtKey string,
	validator *validator.Validate,
	userRepo repositories.UsersRepo,
	orderPaymentRepo repositories.OrderPaymentRepo,
) *OrderPaymentServImpl {
	return &OrderPaymentServImpl{
		Db:               db,
		JwtKey:           jwtKey,
		Validator:        validator,
		UserRepo:         userRepo,
		OrderPaymentRepo: orderPaymentRepo,
	}
}

func (serv *OrderPaymentServImpl) getSchema(clientID uuid.UUID) (string, error) {
	user, err := serv.UserRepo.GetByUserId(serv.Db, clientID)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}
	return user.Username, nil
}

func (serv *OrderPaymentServImpl) checkRole(accessToken string) error {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Admin", "Client"})
	if err != nil || !ok {
		return fmt.Errorf("access denied")
	}
	return nil
}

func (serv *OrderPaymentServImpl) GetAll(accessToken string, clientID uuid.UUID, pg domains.Pagination) (*pagination.Response, error) {
	if err := serv.checkRole(accessToken); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(clientID)
	if err != nil {
		return nil, err
	}

	payments, total, err := serv.OrderPaymentRepo.GetAll(serv.Db, schema, pg)
	if err != nil {
		log.Printf("[OrderPaymentRepo].GetAll error: %v", err)
		return nil, fmt.Errorf("failed to get order payments")
	}

	responses := resOP.ToOrderPaymentResponses(payments)
	result := pagination.ToResponse(responses, total, pg.Page, pg.Limit)
	return &result, nil
}

func (serv *OrderPaymentServImpl) GetByID(accessToken string, clientID uuid.UUID, id uuid.UUID) (*resOP.OrderPaymentResponse, error) {
	if err := serv.checkRole(accessToken); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(clientID)
	if err != nil {
		return nil, err
	}

	payment, err := serv.OrderPaymentRepo.GetByID(serv.Db, schema, id)
	if err != nil {
		log.Printf("[OrderPaymentRepo].GetByID error: %v", err)
		return nil, fmt.Errorf("order payment not found")
	}

	response := resOP.ToOrderPaymentResponse(*payment)
	return &response, nil
}

func (serv *OrderPaymentServImpl) UpdateStatus(accessToken string, clientID uuid.UUID, id uuid.UUID, request reqOP.UpdatePaymentStatusRequest) error {
	if err := serv.checkRole(accessToken); err != nil {
		return err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return err
	}

	schema, err := serv.getSchema(clientID)
	if err != nil {
		return err
	}

	_, err = serv.OrderPaymentRepo.GetByID(serv.Db, schema, id)
	if err != nil {
		log.Printf("[OrderPaymentRepo].GetByID error: %v", err)
		return fmt.Errorf("order payment not found")
	}

	if err := serv.OrderPaymentRepo.UpdateStatus(serv.Db, schema, id, request.Status); err != nil {
		log.Printf("[OrderPaymentRepo].UpdateStatus error: %v", err)
		return fmt.Errorf("failed to update payment status")
	}

	return nil
}
