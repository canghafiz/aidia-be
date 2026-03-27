package product

import (
	"backend/models/domains"
	"backend/models/responses/pagination"
	"time"
)

type ImageResponse struct {
	ID    string `json:"id"`
	Image string `json:"image"`
}

type DeliveryResponse struct {
	SubGroupName string `json:"sub_group_name"`
	Name         string `json:"name"`
	IsVisible    bool   `json:"is_visible"`
	Description  string `json:"description"`
}

type CategoryResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	IsVisible   bool    `json:"is_visible"`
	Description *string `json:"description"`
}

type Response struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Weight        float64            `json:"weight"`
	Price         float64            `json:"price"`
	OriginalPrice float64            `json:"original_price"`
	Description   *string            `json:"description"`
	IsOutOfStock  bool               `json:"is_out_of_stock"`
	IsActive      bool               `json:"is_active"`
	Images        []ImageResponse    `json:"images"`
	Delivery      *DeliveryResponse  `json:"delivery"`
	Categories    []CategoryResponse `json:"categories"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

func ToProductImageResponse(img domains.ProductImage) ImageResponse {
	return ImageResponse{
		ID:    img.ID.String(),
		Image: img.Image,
	}
}

func ToProductResponse(
	p domains.Product,
	delivery *domains.DeliverySetting,
	categories []domains.ProductCategory,
) Response {
	var images []ImageResponse
	for _, img := range p.Images {
		images = append(images, ToProductImageResponse(img))
	}

	var deliveryRes *DeliveryResponse
	if delivery != nil {
		deliveryRes = &DeliveryResponse{
			SubGroupName: delivery.SubGroupName,
			Name:         delivery.Name,
			IsVisible:    delivery.IsVisible,
			Description:  delivery.Description,
		}
	}

	var categoryRes []CategoryResponse
	for _, c := range categories {
		categoryRes = append(categoryRes, CategoryResponse{
			ID:          c.ID.String(),
			Name:        c.Name,
			IsVisible:   c.IsVisible,
			Description: c.Description,
		})
	}

	return Response{
		ID:            p.ID.String(),
		Name:          p.Name,
		Weight:        p.Weight,
		Price:         p.Price,
		OriginalPrice: p.OriginalPrice,
		Description:   p.Description,
		IsOutOfStock:  p.IsOutOfStock,
		IsActive:      p.IsActive,
		Images:        images,
		Delivery:      deliveryRes,
		Categories:    categoryRes,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func ToProductPaginationResponse(
	products []domains.Product,
	deliveries map[string]*domains.DeliverySetting,
	categoriesMap map[string][]domains.ProductCategory,
	total int,
	page, limit int,
) pagination.Response {
	var responses []Response
	for _, p := range products {
		delivery := deliveries[p.DeliveryID.String()]
		categories := categoriesMap[p.ID.String()]
		responses = append(responses, ToProductResponse(p, delivery, categories))
	}
	return pagination.ToResponse(responses, total, page, limit)
}
