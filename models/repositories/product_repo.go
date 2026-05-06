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

	// Tag DTO
	CreateTagDtos(db *gorm.DB, schema string, dtos []domains.ProductTagDto) error
	DeleteTagDtosByProductID(db *gorm.DB, schema string, productID uuid.UUID) error
	GetTagsByProductID(db *gorm.DB, schema string, productID uuid.UUID) ([]domains.ProductTag, error)

	// GetAllByTagIDs returns products that have at least one of the given tag IDs
	GetAllByTagIDs(db *gorm.DB, schema string, tagIDs []uuid.UUID, pagination domains.Pagination) ([]domains.Product, int, error)

	// DecrementQuantity atomically reduces product_quantity by qty.
	// Only executes when current stock >= qty. Auto-marks is_out_of_stock when it hits 0.
	DecrementQuantity(db *gorm.DB, schema string, productID uuid.UUID, qty int) error
}
