package services

import (
	"backend/models/domains"
	reqProduct "backend/models/requests/product"
	"backend/models/responses/pagination"
	resProduct "backend/models/responses/product"
	"mime/multipart"

	"github.com/google/uuid"
)

type ProductServ interface {
	Create(userID uuid.UUID, request reqProduct.CreateProductRequest, images []*multipart.FileHeader) error
	Update(userID uuid.UUID, productID uuid.UUID, request reqProduct.UpdateProductRequest, images []*multipart.FileHeader) error
	GetAll(userID uuid.UUID, pg domains.Pagination) (*pagination.Response, error)
	GetByID(userID uuid.UUID, productID uuid.UUID) (*resProduct.Response, error)
	Delete(userID uuid.UUID, productID uuid.UUID) error
}
