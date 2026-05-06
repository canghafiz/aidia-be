package product_tag

import "backend/models/domains"

type CreateProductTagRequest struct {
	Name string `json:"name" validate:"required,max=100"`
}

type UpdateProductTagRequest struct {
	Name string `json:"name" validate:"required,max=100"`
}

func CreateProductTagToDomain(req CreateProductTagRequest) domains.ProductTag {
	return domains.ProductTag{Name: req.Name}
}

func UpdateProductTagToDomain(req UpdateProductTagRequest) domains.ProductTag {
	return domains.ProductTag{Name: req.Name}
}
