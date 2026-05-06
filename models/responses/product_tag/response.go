package product_tag

import "backend/models/domains"

type ProductTagResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func ToProductTagResponse(t domains.ProductTag) ProductTagResponse {
	return ProductTagResponse{ID: t.ID.String(), Name: t.Name}
}

func ToProductTagResponses(tags []domains.ProductTag) []ProductTagResponse {
	var out []ProductTagResponse
	for _, t := range tags {
		out = append(out, ToProductTagResponse(t))
	}
	return out
}
