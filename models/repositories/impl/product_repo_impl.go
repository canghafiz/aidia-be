package impl

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductRepoImpl struct{}

func NewProductRepoImpl() *ProductRepoImpl {
	return &ProductRepoImpl{}
}

func (repo *ProductRepoImpl) Create(db *gorm.DB, schema string, product domains.Product) (*domains.Product, error) {
	if err := db.Table(schema + ".product").Create(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func (repo *ProductRepoImpl) Update(db *gorm.DB, schema string, product domains.Product) error {
	return db.Table(schema+".product").
		Where("id = ?", product.ID).
		Updates(map[string]interface{}{
			"name":            product.Name,
			"code":            product.Code,
			"weight":          product.Weight,
			"price":           product.Price,
			"original_price":  product.OriginalPrice,
			"description":     product.Description,
			"delivery_id":     product.DeliveryID,
			"is_out_of_stock": product.IsOutOfStock,
			"is_active":       product.IsActive,
		}).Error
}

func (repo *ProductRepoImpl) GetAll(db *gorm.DB, schema string, pagination domains.Pagination) ([]domains.Product, int, error) {
	var products []domains.Product
	var total int64

	if err := db.Table(schema + ".product").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Raw(`
		SELECT * FROM `+schema+`.product
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, pagination.Limit, pagination.Offset()).
		Scan(&products).Error; err != nil {
		return nil, 0, err
	}

	for i, p := range products {
		var images []domains.ProductImage
		if err := db.Raw(`
			SELECT * FROM `+schema+`.product_image
			WHERE product_id = ? AND is_active = true`, p.ID).
			Scan(&images).Error; err != nil {
			return nil, 0, err
		}
		products[i].Images = images
	}

	return products, int(total), nil
}

func (repo *ProductRepoImpl) GetByID(db *gorm.DB, schema string, id uuid.UUID) (*domains.Product, error) {
	var product domains.Product
	if err := db.Raw(`
		SELECT * FROM `+schema+`.product WHERE id = ?`, id).
		Scan(&product).Error; err != nil {
		return nil, err
	}
	if product.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}

	var images []domains.ProductImage
	if err := db.Raw(`
		SELECT * FROM `+schema+`.product_image
		WHERE product_id = ? AND is_active = true`, id).
		Scan(&images).Error; err != nil {
		return nil, err
	}
	product.Images = images

	return &product, nil
}

func (repo *ProductRepoImpl) Delete(db *gorm.DB, schema string, id uuid.UUID) error {
	return db.Table(schema+".product").
		Where("id = ?", id).
		Delete(&domains.Product{}).Error
}

func (repo *ProductRepoImpl) CreateImages(db *gorm.DB, schema string, images []domains.ProductImage) error {
	return db.Table(schema + ".product_image").Create(&images).Error
}

func (repo *ProductRepoImpl) DeleteImagesByProductID(db *gorm.DB, schema string, productID uuid.UUID) error {
	return db.Table(schema+".product_image").
		Where("product_id = ?", productID).
		Delete(&domains.ProductImage{}).Error
}

func (repo *ProductRepoImpl) CreateCategoryDtos(db *gorm.DB, schema string, dtos []domains.ProductCategoryDto) error {
	return db.Table(schema + ".product_category_dto").Create(&dtos).Error
}

func (repo *ProductRepoImpl) DeleteCategoryDtosByProductID(db *gorm.DB, schema string, productID uuid.UUID) error {
	return db.Table(schema+".product_category_dto").
		Where("product_id = ?", productID).
		Delete(&domains.ProductCategoryDto{}).Error
}

func (repo *ProductRepoImpl) GetCategoriesByProductID(db *gorm.DB, schema string, productID uuid.UUID) ([]domains.ProductCategory, error) {
	var categories []domains.ProductCategory
	if err := db.Raw(`
		SELECT pc.* FROM `+schema+`.product_category pc
		JOIN `+schema+`.product_category_dto pcd ON pcd.category_id = pc.id
		WHERE pcd.product_id = ?`, productID).
		Scan(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}
