package impl

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductCategoryRepoImpl struct{}

func NewProductCategoryRepoImpl() *ProductCategoryRepoImpl {
	return &ProductCategoryRepoImpl{}
}

func (repo *ProductCategoryRepoImpl) Create(db *gorm.DB, schema string, category domains.ProductCategory) error {
	return db.Table(schema + ".product_category").Create(&category).Error
}

func (repo *ProductCategoryRepoImpl) Update(db *gorm.DB, schema string, category domains.ProductCategory) error {
	return db.Table(schema+".product_category").
		Where("id = ?", category.ID).
		Updates(map[string]interface{}{
			"name":        category.Name,
			"is_visible":  category.IsVisible,
			"description": category.Description,
		}).Error
}

func (repo *ProductCategoryRepoImpl) GetByName(db *gorm.DB, schema string, name string) (*domains.ProductCategory, error) {
	var category domains.ProductCategory
	if err := db.Table(schema+".product_category").
		Where("name = ?", name).
		First(&category).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (repo *ProductCategoryRepoImpl) GetAll(db *gorm.DB, schema string, isVisible bool) ([]domains.ProductCategory, error) {
	var categories []domains.ProductCategory
	if err := db.Table(schema+".product_category").
		Where("is_visible = ?", isVisible).
		Find(&categories).Error; err != nil {
		return nil, err
	}

	for i, cat := range categories {
		var products []domains.Product
		if err := db.Raw(`
        SELECT p.* FROM `+schema+`.product p
        JOIN `+schema+`.product_category_dto pcd ON pcd.product_id = p.id
        WHERE pcd.category_id = ?`, cat.ID).
			Scan(&products).Error; err != nil {
			return nil, err
		}
		categories[i].Products = products
	}

	return categories, nil
}

func (repo *ProductCategoryRepoImpl) GetByIdWithProducts(db *gorm.DB, schema string, id uuid.UUID) (*domains.ProductCategory, error) {
	var category domains.ProductCategory
	if err := db.Table(schema+".product_category").
		Where("id = ?", id).
		First(&category).Error; err != nil {
		return nil, err
	}

	var products []domains.Product
	if err := db.Raw(`
    SELECT p.* FROM `+schema+`.product p
    JOIN `+schema+`.product_category_dto pcd ON pcd.product_id = p.id
    WHERE pcd.category_id = ?`, category.ID).
		Scan(&products).Error; err != nil {
		return nil, err
	}

	// Load images per product
	for i, p := range products {
		var images []domains.ProductImage
		if err := db.Table(schema+".product_image").
			Where("product_id = ? AND is_active = ?", p.ID, true).
			Find(&images).Error; err != nil {
			return nil, err
		}
		products[i].Images = images
	}

	category.Products = products
	return &category, nil
}

func (repo *ProductCategoryRepoImpl) GetById(db *gorm.DB, schema string, id uuid.UUID) (*domains.ProductCategory, error) {
	var category domains.ProductCategory
	if err := db.Table(schema+".product_category").
		Where("id = ?", id).
		First(&category).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (repo *ProductCategoryRepoImpl) Delete(db *gorm.DB, schema string, id uuid.UUID) error {
	return db.Table(schema+".product_category").
		Where("id = ?", id).
		Delete(&domains.ProductCategory{}).Error
}
