package services

import (
	"backend/models/requests/product_tag"
	resTag "backend/models/responses/product_tag"

	"github.com/google/uuid"
)

type ProductTagServ interface {
	Create(userID uuid.UUID, req product_tag.CreateProductTagRequest) error
	Update(userID uuid.UUID, id uuid.UUID, req product_tag.UpdateProductTagRequest) error
	GetAll(userID uuid.UUID) ([]resTag.ProductTagResponse, error)
	GetByID(userID uuid.UUID, id uuid.UUID) (*resTag.ProductTagResponse, error)
	Delete(userID uuid.UUID, id uuid.UUID) error
}
