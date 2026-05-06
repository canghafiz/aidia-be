package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductTagRepo interface {
	Create(db *gorm.DB, schema string, tag domains.ProductTag) error
	Update(db *gorm.DB, schema string, tag domains.ProductTag) error
	GetAll(db *gorm.DB, schema string) ([]domains.ProductTag, error)
	GetByID(db *gorm.DB, schema string, id uuid.UUID) (*domains.ProductTag, error)
	GetByName(db *gorm.DB, schema string, name string) (*domains.ProductTag, error)
	Delete(db *gorm.DB, schema string, id uuid.UUID) error
	GetTagsByProductID(db *gorm.DB, schema string, productID uuid.UUID) ([]domains.ProductTag, error)
	CreateTagDtos(db *gorm.DB, schema string, dtos []domains.ProductTagDto) error
	DeleteTagDtosByProductID(db *gorm.DB, schema string, productID uuid.UUID) error
}
