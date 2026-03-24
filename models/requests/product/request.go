package product

import (
	"backend/models/domains"

	"github.com/google/uuid"
)

type CreateProductRequest struct {
	Name          string      `json:"name"           validate:"required,max=150"`
	Code          string      `json:"code"           validate:"required,max=30"`
	Weight        float64     `json:"weight"         validate:"required,min=0"`
	Price         float64     `json:"price"          validate:"required,min=0"`
	OriginalPrice float64     `json:"original_price" validate:"required,min=0"`
	Description   *string     `json:"description"    validate:"omitempty"`
	DeliveryID    uuid.UUID   `json:"delivery_sub_group_name"    validate:"required"`
	IsOutOfStock  bool        `json:"is_out_of_stock"`
	CategoryIDs   []uuid.UUID `json:"category_ids"   validate:"omitempty"`
}

type UpdateProductRequest struct {
	Name          string      `json:"name"           validate:"required,max=150"`
	Code          string      `json:"code"           validate:"required,max=30"`
	Weight        float64     `json:"weight"         validate:"required,min=0"`
	Price         float64     `json:"price"          validate:"required,min=0"`
	OriginalPrice float64     `json:"original_price" validate:"required,min=0"`
	Description   *string     `json:"description"    validate:"omitempty"`
	DeliveryID    uuid.UUID   `json:"delivery_sub_group_name"    validate:"required"`
	IsOutOfStock  bool        `json:"is_out_of_stock"`
	IsActive      bool        `json:"is_active"`
	CategoryIDs   []uuid.UUID `json:"category_ids"   validate:"omitempty"`
}

func CreateProductToDomain(req CreateProductRequest) domains.Product {
	return domains.Product{
		Name:          req.Name,
		Code:          req.Code,
		Weight:        req.Weight,
		Price:         req.Price,
		OriginalPrice: req.OriginalPrice,
		Description:   req.Description,
		DeliveryID:    req.DeliveryID,
		IsOutOfStock:  req.IsOutOfStock,
		IsActive:      true,
	}
}

func UpdateProductToDomain(req UpdateProductRequest) domains.Product {
	return domains.Product{
		Name:          req.Name,
		Code:          req.Code,
		Weight:        req.Weight,
		Price:         req.Price,
		OriginalPrice: req.OriginalPrice,
		Description:   req.Description,
		DeliveryID:    req.DeliveryID,
		IsOutOfStock:  req.IsOutOfStock,
		IsActive:      req.IsActive,
	}
}
