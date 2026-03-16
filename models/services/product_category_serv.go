package services

import (
	"backend/models/requests/product_category"
	resCat "backend/models/responses/product_category"

	"github.com/google/uuid"
)

type ProductCategoryServ interface {
	Create(userId uuid.UUID, request product_category.CreateProductCategoryRequest) error
	Update(userId uuid.UUID, id uuid.UUID, request product_category.UpdateProductCategoryRequest) error
	GetAll(userId uuid.UUID, isVisible bool) ([]resCat.ProductCategoryResponse, error)
	GetByID(userId uuid.UUID, id uuid.UUID) (*resCat.ProductCategoryDetailResponse, error)
	Delete(userId uuid.UUID, id uuid.UUID) error
}
