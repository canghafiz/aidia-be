package impl

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlanRepoImpl struct {
}

func NewPlanRepoImpl() *PlanRepoImpl {
	return &PlanRepoImpl{}
}

func (repo *PlanRepoImpl) Create(db *gorm.DB, plan domains.Plan) error {
	return db.Create(&plan).Error
}

func (repo *PlanRepoImpl) ToggleIsActive(db *gorm.DB, id uuid.UUID, isActive bool) error {
	return db.Model(&domains.Plan{}).
		Where("id = ?", id).
		Update("is_active", isActive).Error
}

func (repo *PlanRepoImpl) Update(db *gorm.DB, plan domains.Plan) error {
	updates := map[string]interface{}{}

	if plan.Name != "" {
		updates["name"] = plan.Name
	}
	if plan.Duration != 0 {
		updates["duration"] = plan.Duration
	}
	if plan.Price != 0 {
		updates["price"] = plan.Price
	}

	updates["is_month"] = plan.IsMonth
	updates["is_active"] = plan.IsActive

	return db.Model(&domains.Plan{}).
		Where("id = ?", plan.ID).
		Updates(updates).Error
}

func (repo *PlanRepoImpl) GetAll(db *gorm.DB, pagination domains.Pagination) ([]domains.Plan, int, error) {
	var plans []domains.Plan
	var total int64

	query := db.Model(&domains.Plan{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Find(&plans).Error; err != nil {
		return nil, 0, err
	}

	return plans, int(total), nil
}

func (repo *PlanRepoImpl) GetAllPublic(db *gorm.DB) ([]domains.Plan, int, error) {
	var plans []domains.Plan
	var total int64

	query := db.Model(&domains.Plan{}).Where("is_active = ?", true)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Find(&plans).Error; err != nil {
		return nil, 0, err
	}

	return plans, int(total), nil
}

func (repo *PlanRepoImpl) GetById(db *gorm.DB, id uuid.UUID) (*domains.Plan, error) {
	var plan domains.Plan
	err := db.Where("id = ?", id).First(&plan).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (repo *PlanRepoImpl) GetByIdPublic(db *gorm.DB, id uuid.UUID) (*domains.Plan, error) {
	var plan domains.Plan
	if err := db.Where("id = ? AND is_active = ?", id, true).First(&plan).Error; err != nil {
		return nil, err
	}
	return &plan, nil
}

func (repo *PlanRepoImpl) Delete(db *gorm.DB, plan domains.Plan) error {
	return db.Where("id = ?", plan.ID).Delete(&domains.Plan{}).Error
}
