package impl

import (
	"backend/helpers"
	"backend/models/repositories"
	"backend/models/requests/product_category"
	resCat "backend/models/responses/product_category"
	"fmt"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductCategoryServImpl struct {
	Db                  *gorm.DB
	JwtKey              string
	Validator           *validator.Validate
	UserRepo            repositories.UsersRepo
	ProductCategoryRepo repositories.ProductCategoryRepo
}

func NewProductCategoryServImpl(db *gorm.DB, jwtKey string, validator *validator.Validate, userRepo repositories.UsersRepo, productCategoryRepo repositories.ProductCategoryRepo) *ProductCategoryServImpl {
	return &ProductCategoryServImpl{Db: db, JwtKey: jwtKey, Validator: validator, UserRepo: userRepo, ProductCategoryRepo: productCategoryRepo}
}

func (serv *ProductCategoryServImpl) getSchema(userID uuid.UUID) (string, error) {
	user, err := serv.UserRepo.GetByUserId(serv.Db, userID)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}
	return user.Username, nil
}

func (serv *ProductCategoryServImpl) checkClientRole(userID uuid.UUID) error {
	role, err := serv.UserRepo.GetUserRole(serv.Db, userID)
	if err != nil {
		return fmt.Errorf("failed to get user role")
	}
	if role != "Client" {
		return fmt.Errorf("access denied")
	}
	return nil
}

func (serv *ProductCategoryServImpl) Create(userID uuid.UUID, request product_category.CreateProductCategoryRequest) error {
	if err := serv.checkClientRole(userID); err != nil {
		return err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return err
	}

	domain := product_category.CreateProductCategoryToDomain(request)
	if err := serv.ProductCategoryRepo.Create(serv.Db, schema, domain); err != nil {
		log.Printf("[ProductCategoryRepo].Create error: %v", err)
		return fmt.Errorf("failed to create product category")
	}

	return nil
}

func (serv *ProductCategoryServImpl) Update(userID uuid.UUID, id uuid.UUID, request product_category.UpdateProductCategoryRequest) error {
	if err := serv.checkClientRole(userID); err != nil {
		return err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return err
	}

	_, err = serv.ProductCategoryRepo.GetById(serv.Db, schema, id)
	if err != nil {
		log.Printf("[ProductCategoryRepo].GetById error: %v", err)
		return fmt.Errorf("product category not found")
	}

	domain := product_category.UpdateProductCategoryToDomain(request)
	domain.ID = id
	if err := serv.ProductCategoryRepo.Update(serv.Db, schema, domain); err != nil {
		log.Printf("[ProductCategoryRepo].Update error: %v", err)
		return fmt.Errorf("failed to update product category")
	}

	return nil
}

func (serv *ProductCategoryServImpl) GetAll(userID uuid.UUID, isVisible bool) ([]resCat.ProductCategoryResponse, error) {
	if err := serv.checkClientRole(userID); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return nil, err
	}

	categories, err := serv.ProductCategoryRepo.GetAll(serv.Db, schema, isVisible)
	if err != nil {
		log.Printf("[ProductCategoryRepo].GetAll error: %v", err)
		return nil, fmt.Errorf("failed to get product categories")
	}

	return resCat.ToProductCategoryResponses(categories), nil
}

func (serv *ProductCategoryServImpl) GetByID(userID uuid.UUID, id uuid.UUID) (*resCat.ProductCategoryDetailResponse, error) {
	if err := serv.checkClientRole(userID); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return nil, err
	}

	category, err := serv.ProductCategoryRepo.GetByIdWithProducts(serv.Db, schema, id)
	if err != nil {
		log.Printf("[ProductCategoryRepo].GetByIdWithProducts error: %v", err)
		return nil, fmt.Errorf("product category not found")
	}

	response := resCat.ToProductCategoryDetailResponse(*category)
	return &response, nil
}

func (serv *ProductCategoryServImpl) Delete(userID uuid.UUID, id uuid.UUID) error {
	if err := serv.checkClientRole(userID); err != nil {
		return err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return err
	}

	_, err = serv.ProductCategoryRepo.GetById(serv.Db, schema, id)
	if err != nil {
		log.Printf("[ProductCategoryRepo].GetById error: %v", err)
		return fmt.Errorf("product category not found")
	}

	if err := serv.ProductCategoryRepo.Delete(serv.Db, schema, id); err != nil {
		log.Printf("[ProductCategoryRepo].Delete error: %v", err)
		return fmt.Errorf("failed to delete product category")
	}

	return nil
}
