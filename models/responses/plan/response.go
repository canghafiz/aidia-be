package plan

import (
	"backend/models/domains"
	"time"
)

type Response struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IsMonth   bool      `json:"is_month"`
	Duration  int       `json:"duration"`
	Price     float64   `json:"price"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PublicResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	IsMonth  bool    `json:"is_month"`
	Duration int     `json:"duration"`
	Price    float64 `json:"price"`
}

func ToResponse(p domains.Plan) Response {
	return Response{
		ID:        p.ID.String(),
		Name:      p.Name,
		IsMonth:   p.IsMonth,
		Duration:  p.Duration,
		Price:     p.Price,
		IsActive:  p.IsActive,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func ToPublicResponse(p domains.Plan) PublicResponse {
	return PublicResponse{
		ID:       p.ID.String(),
		Name:     p.Name,
		IsMonth:  p.IsMonth,
		Duration: p.Duration,
		Price:    p.Price,
	}
}

func ToResponses(plans []domains.Plan) []Response {
	var responses []Response
	for _, p := range plans {
		responses = append(responses, ToResponse(p))
	}
	return responses
}

func ToPublicResponses(plans []domains.Plan) []PublicResponse {
	var responses []PublicResponse
	for _, p := range plans {
		responses = append(responses, ToPublicResponse(p))
	}
	return responses
}
