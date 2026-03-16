package plan

import "backend/models/domains"

type UpdateRequest struct {
	Name     string  `json:"name"      validate:"omitempty,min=1,max=100"`
	IsMonth  *bool   `json:"is_month"  validate:"omitempty"`
	Duration int     `json:"duration"  validate:"omitempty,min=1"`
	Price    float64 `json:"price"     validate:"omitempty,min=0"`
	IsActive *bool   `json:"is_active" validate:"omitempty"`
}

func UpdateRequestToDomain(req UpdateRequest) domains.Plan {
	plan := domains.Plan{
		Name:     req.Name,
		Duration: req.Duration,
		Price:    req.Price,
	}

	if req.IsMonth != nil {
		plan.IsMonth = *req.IsMonth
	}
	if req.IsActive != nil {
		plan.IsActive = *req.IsActive
	}

	return plan
}
