package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductCategoryRepo interface {
	Create(db *gorm.DB, schema string, category domains.ProductCategory) error
	Update(db *gorm.DB, schema string, category domains.ProductCategory) error
	GetByName(db *gorm.DB, schema string, name string) (*domains.ProductCategory, error)
	GetAll(db *gorm.DB, schema string, isVisible bool) ([]domains.ProductCategory, error)
	GetById(db *gorm.DB, schema string, id uuid.UUID) (*domains.ProductCategory, error)
	GetByIdWithProducts(db *gorm.DB, schema string, id uuid.UUID) (*domains.ProductCategory, error)
	Delete(db *gorm.DB, schema string, id uuid.UUID) error
}
