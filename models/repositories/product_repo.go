package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductRepo interface {
	Create(db *gorm.DB, schema string, product domains.Product) (*domains.Product, error)
	Update(db *gorm.DB, schema string, product domains.Product) error
	GetAll(db *gorm.DB, schema string, pagination domains.Pagination) ([]domains.Product, int, error)
	GetByID(db *gorm.DB, schema string, id uuid.UUID) (*domains.Product, error)
	Delete(db *gorm.DB, schema string, id uuid.UUID) error

	// CreateImages Image
	CreateImages(db *gorm.DB, schema string, images []domains.ProductImage) error
	DeleteImagesByProductID(db *gorm.DB, schema string, productID uuid.UUID) error

	// CreateCategoryDtos Category DTO
	CreateCategoryDtos(db *gorm.DB, schema string, dtos []domains.ProductCategoryDto) error
	DeleteCategoryDtosByProductID(db *gorm.DB, schema string, productID uuid.UUID) error

	// GetCategoriesByProductID Get categories by product ID
	GetCategoriesByProductID(db *gorm.DB, schema string, productID uuid.UUID) ([]domains.ProductCategory, error)
}
