package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	req "backend/models/requests/customer"
	res "backend/models/responses/customer"
	"backend/models/responses/pagination"
	"fmt"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CustomerServImpl struct {
	Db           *gorm.DB
	Validator    *validator.Validate
	UserRepo     repositories.UsersRepo
	CustomerRepo repositories.CustomerRepo
	JwtKey       string
}

func NewCustomerServImpl(db *gorm.DB, validator *validator.Validate, userRepo repositories.UsersRepo, customerRepo repositories.CustomerRepo, jwtKey string) *CustomerServImpl {
	return &CustomerServImpl{Db: db, Validator: validator, UserRepo: userRepo, CustomerRepo: customerRepo, JwtKey: jwtKey}
}

func (serv *CustomerServImpl) getSchema(clientID uuid.UUID) (string, error) {
	user, err := serv.UserRepo.GetByUserId(serv.Db, clientID)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}
	return user.Username, nil
}

func (serv *CustomerServImpl) checkClientRole(clientID uuid.UUID) error {
	role, err := serv.UserRepo.GetUserRole(serv.Db, clientID)
	if err != nil {
		return fmt.Errorf("failed to get user role")
	}
	if role != "Client" {
		return fmt.Errorf("access denied")
	}
	return nil
}

func (serv *CustomerServImpl) checkRole(accessToken string) error {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Admin"})
	if err != nil || !ok {
		return fmt.Errorf("access denied")
	}
	return nil
}

func (serv *CustomerServImpl) Create(accessToken string, clientID uuid.UUID, request req.CreateCustomerRequest) (*res.Response, error) {
	if err := serv.checkRole(accessToken); err != nil {
		return nil, err
	}

	if err := serv.checkClientRole(clientID); err != nil {
		return nil, err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(clientID)
	if err != nil {
		return nil, err
	}

	existing, err := serv.CustomerRepo.GetByPhone(serv.Db, schema, request.PhoneCountryCode, request.PhoneNumber)
	if err == nil && existing != nil {
		response := res.ToResponse(*existing)
		return &response, nil
	}

	domain := req.CreateCustomerToDomain(request)
	customer, err := serv.CustomerRepo.Create(serv.Db, schema, domain)
	if err != nil {
		log.Printf("[CustomerRepo].Create error: %v", err)
		return nil, fmt.Errorf("failed to create customer")
	}

	response := res.ToResponse(*customer)
	return &response, nil
}

func (serv *CustomerServImpl) GetAll(accessToken string, clientID uuid.UUID, pg domains.Pagination) (*pagination.Response, error) {
	if err := serv.checkRole(accessToken); err != nil {
		return nil, err
	}

	if err := serv.checkClientRole(clientID); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(clientID)
	if err != nil {
		return nil, err
	}

	customers, total, err := serv.CustomerRepo.GetAll(serv.Db, schema, pg)
	if err != nil {
		log.Printf("[CustomerRepo].GetAll error: %v", err)
		return nil, fmt.Errorf("failed to get customers")
	}

	responses := res.ToResponses(customers)
	result := pagination.ToResponse(responses, total, pg.Page, pg.Limit)
	return &result, nil
}

func (serv *CustomerServImpl) GetByID(accessToken string, clientID uuid.UUID, id int) (*res.Response, error) {
	if err := serv.checkRole(accessToken); err != nil {
		return nil, err
	}

	if err := serv.checkClientRole(clientID); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(clientID)
	if err != nil {
		return nil, err
	}

	customer, err := serv.CustomerRepo.GetByID(serv.Db, schema, id)
	if err != nil {
		log.Printf("[CustomerRepo].GetByID error: %v", err)
		return nil, fmt.Errorf("customer not found")
	}

	response := res.ToResponse(*customer)
	return &response, nil
}
