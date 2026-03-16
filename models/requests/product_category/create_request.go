package product_category

import "backend/models/domains"

type CreateProductCategoryRequest struct {
	Name        string  `json:"name"        validate:"required,max=100"`
	IsVisible   bool    `json:"is_visible"`
	Description *string `json:"description" validate:"omitempty,max=100"`
}

func CreateProductCategoryToDomain(request CreateProductCategoryRequest) domains.ProductCategory {
	return domains.ProductCategory{
		Name:        request.Name,
		IsVisible:   request.IsVisible,
		Description: request.Description,
	}
}
