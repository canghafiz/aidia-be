package plan

import "backend/models/domains"

type CreateRequest struct {
	Name     string  `json:"name"      validate:"required,min=1,max=100"`
	IsMonth  bool    `json:"is_month"  validate:"required"`
	Duration int     `json:"duration"  validate:"required,min=1"`
	Price    float64 `json:"price"     validate:"required,min=0"`
	IsActive bool    `json:"is_active"`
}

func CreateRequestToDomain(req CreateRequest) domains.Plan {
	return domains.Plan{
		Name:     req.Name,
		IsMonth:  req.IsMonth,
		Duration: req.Duration,
		Price:    req.Price,
		IsActive: req.IsActive,
	}
}
