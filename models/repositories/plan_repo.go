package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlanRepo interface {
	Create(db *gorm.DB, plan domains.Plan) error
	ToggleIsActive(db *gorm.DB, id uuid.UUID, isActive bool) error
	Update(db *gorm.DB, plan domains.Plan) error
	GetAll(db *gorm.DB, pagination domains.Pagination) (plans []domains.Plan, totalData int, err error)
	GetAllPublic(db *gorm.DB) (plans []domains.Plan, totalData int, err error)
	GetById(db *gorm.DB, id uuid.UUID) (plan *domains.Plan, err error)
	GetByIdPublic(db *gorm.DB, id uuid.UUID) (plan *domains.Plan, err error)
	Delete(db *gorm.DB, plan domains.Plan) error
}
