package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type KitchenOrderRepo interface {
	GetAll(db *gorm.DB, schema string) ([]domains.KitchenOrder, error)
	GetByStatus(db *gorm.DB, schema string, status domains.KitchenStatus) ([]domains.KitchenOrder, error)
	GetByID(db *gorm.DB, schema string, id uuid.UUID) (*domains.KitchenOrder, error)
	UpdateStatus(db *gorm.DB, schema string, id uuid.UUID, status domains.KitchenStatus) error
}
