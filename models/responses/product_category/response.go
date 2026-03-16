package product_category

import (
	"backend/models/domains"
	"time"
)

// ============================================================
// PRODUCT CATEGORY — LIST
// ============================================================

type ProductCategoryResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	IsVisible    bool   `json:"is_visible"`
	TotalProduct int    `json:"total_product"`
}

// ============================================================
// PRODUCT CATEGORY — DETAIL
// ============================================================

type ProductCategoryDetailResponse struct {
	ID           string                       `json:"id"`
	Name         string                       `json:"name"`
	IsVisible    bool                         `json:"is_visible"`
	Description  *string                      `json:"description"`
	TotalProduct int                          `json:"total_product"`
	Products     []ProductForCategoryResponse `json:"products"`
	CreatedAt    time.Time                    `json:"created_at"`
	UpdatedAt    time.Time                    `json:"updated_at"`
}

// ============================================================
// PRODUCT — FOR CATEGORY (dipakai di detail category)
// ============================================================

type ProductForCategoryResponse struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	IsVisible bool                   `json:"is_visible"`
	Images    []ProductImageResponse `json:"images"`
}

type ProductImageResponse struct {
	ID    string `json:"id"`
	Image string `json:"image"`
}

// ============================================================
// MAPPER
// ============================================================

func ToProductCategoryResponse(c domains.ProductCategory) ProductCategoryResponse {
	return ProductCategoryResponse{
		ID:           c.ID.String(),
		Name:         c.Name,
		IsVisible:    c.IsVisible,
		TotalProduct: len(c.Products),
	}
}

func ToProductCategoryResponses(categories []domains.ProductCategory) []ProductCategoryResponse {
	var responses []ProductCategoryResponse
	for _, c := range categories {
		responses = append(responses, ToProductCategoryResponse(c))
	}
	return responses
}

func ToProductCategoryDetailResponse(c domains.ProductCategory) ProductCategoryDetailResponse {
	var products []ProductForCategoryResponse
	for _, p := range c.Products {
		var images []ProductImageResponse
		for _, img := range p.Images {
			images = append(images, ProductImageResponse{
				ID:    img.ID.String(),
				Image: img.Image,
			})
		}
		products = append(products, ProductForCategoryResponse{
			ID:        p.ID.String(),
			Name:      p.Name,
			IsVisible: p.IsActive,
			Images:    images,
		})
	}

	return ProductCategoryDetailResponse{
		ID:           c.ID.String(),
		Name:         c.Name,
		IsVisible:    c.IsVisible,
		Description:  c.Description,
		TotalProduct: len(c.Products),
		Products:     products,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}
