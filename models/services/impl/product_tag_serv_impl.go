package impl

import (
	"backend/helpers"
	"backend/models/repositories"
	reqTag "backend/models/requests/product_tag"
	resTag "backend/models/responses/product_tag"
	"fmt"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductTagServImpl struct {
	Db             *gorm.DB
	Validator      *validator.Validate
	UserRepo       repositories.UsersRepo
	ProductTagRepo repositories.ProductTagRepo
}

func NewProductTagServImpl(db *gorm.DB, validator *validator.Validate, userRepo repositories.UsersRepo, productTagRepo repositories.ProductTagRepo) *ProductTagServImpl {
	return &ProductTagServImpl{Db: db, Validator: validator, UserRepo: userRepo, ProductTagRepo: productTagRepo}
}

func (s *ProductTagServImpl) getSchema(userID uuid.UUID) (string, error) {
	return helpers.GetSchema(s.Db, s.UserRepo, userID)
}

func (s *ProductTagServImpl) checkClientRole(userID uuid.UUID) error {
	role, err := s.UserRepo.GetUserRole(s.Db, userID)
	if err != nil {
		return fmt.Errorf("failed to get user role")
	}
	if role != "Client" {
		return fmt.Errorf("access denied")
	}
	return nil
}

func (s *ProductTagServImpl) Create(userID uuid.UUID, req reqTag.CreateProductTagRequest) error {
	if err := s.checkClientRole(userID); err != nil {
		return err
	}
	if err := helpers.ErrValidator(req, s.Validator); err != nil {
		return err
	}
	schema, err := s.getSchema(userID)
	if err != nil {
		return err
	}
	if existing, _ := s.ProductTagRepo.GetByName(s.Db, schema, req.Name); existing != nil {
		return fmt.Errorf("tag name already exists")
	}
	domain := reqTag.CreateProductTagToDomain(req)
	if err := s.ProductTagRepo.Create(s.Db, schema, domain); err != nil {
		log.Printf("[ProductTagRepo].Create error: %v", err)
		return fmt.Errorf("failed to create product tag")
	}
	return nil
}

func (s *ProductTagServImpl) Update(userID uuid.UUID, id uuid.UUID, req reqTag.UpdateProductTagRequest) error {
	if err := s.checkClientRole(userID); err != nil {
		return err
	}
	if err := helpers.ErrValidator(req, s.Validator); err != nil {
		return err
	}
	schema, err := s.getSchema(userID)
	if err != nil {
		return err
	}
	if _, err := s.ProductTagRepo.GetByID(s.Db, schema, id); err != nil {
		return fmt.Errorf("product tag not found")
	}
	if existing, _ := s.ProductTagRepo.GetByName(s.Db, schema, req.Name); existing != nil && existing.ID != id {
		return fmt.Errorf("tag name already exists")
	}
	domain := reqTag.UpdateProductTagToDomain(req)
	domain.ID = id
	if err := s.ProductTagRepo.Update(s.Db, schema, domain); err != nil {
		log.Printf("[ProductTagRepo].Update error: %v", err)
		return fmt.Errorf("failed to update product tag")
	}
	return nil
}

func (s *ProductTagServImpl) GetAll(userID uuid.UUID) ([]resTag.ProductTagResponse, error) {
	if err := s.checkClientRole(userID); err != nil {
		return nil, err
	}
	schema, err := s.getSchema(userID)
	if err != nil {
		return nil, err
	}
	tags, err := s.ProductTagRepo.GetAll(s.Db, schema)
	if err != nil {
		log.Printf("[ProductTagRepo].GetAll error: %v", err)
		return nil, fmt.Errorf("failed to get product tags")
	}
	return resTag.ToProductTagResponses(tags), nil
}

func (s *ProductTagServImpl) GetByID(userID uuid.UUID, id uuid.UUID) (*resTag.ProductTagResponse, error) {
	if err := s.checkClientRole(userID); err != nil {
		return nil, err
	}
	schema, err := s.getSchema(userID)
	if err != nil {
		return nil, err
	}
	tag, err := s.ProductTagRepo.GetByID(s.Db, schema, id)
	if err != nil {
		return nil, fmt.Errorf("product tag not found")
	}
	r := resTag.ToProductTagResponse(*tag)
	return &r, nil
}

func (s *ProductTagServImpl) Delete(userID uuid.UUID, id uuid.UUID) error {
	if err := s.checkClientRole(userID); err != nil {
		return err
	}
	schema, err := s.getSchema(userID)
	if err != nil {
		return err
	}
	if _, err := s.ProductTagRepo.GetByID(s.Db, schema, id); err != nil {
		return fmt.Errorf("product tag not found")
	}
	if err := s.ProductTagRepo.Delete(s.Db, schema, id); err != nil {
		log.Printf("[ProductTagRepo].Delete error: %v", err)
		return fmt.Errorf("failed to delete product tag")
	}
	return nil
}
