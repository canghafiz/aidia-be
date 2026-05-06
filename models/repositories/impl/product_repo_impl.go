package impl

import (
	"backend/models/domains"
	"fmt"
	"strings"

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
			"name":             product.Name,
			"weight":           product.Weight,
			"price":            product.Price,
			"original_price":   product.OriginalPrice,
			"description":      product.Description,
			"delivery_id":      product.DeliveryID,
			"is_out_of_stock":  product.IsOutOfStock,
			"product_quantity": product.ProductQuantity,
			"is_active":        product.IsActive,
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

func (repo *ProductRepoImpl) CreateTagDtos(db *gorm.DB, schema string, dtos []domains.ProductTagDto) error {
	return db.Table(schema + ".product_tag_dto").Create(&dtos).Error
}

func (repo *ProductRepoImpl) DeleteTagDtosByProductID(db *gorm.DB, schema string, productID uuid.UUID) error {
	return db.Table(schema+".product_tag_dto").
		Where("product_id = ?", productID).
		Delete(&domains.ProductTagDto{}).Error
}

func (repo *ProductRepoImpl) GetTagsByProductID(db *gorm.DB, schema string, productID uuid.UUID) ([]domains.ProductTag, error) {
	var tags []domains.ProductTag
	if err := db.Raw(`
		SELECT pt.* FROM `+schema+`.product_tag pt
		JOIN `+schema+`.product_tag_dto ptd ON ptd.tag_id = pt.id
		WHERE ptd.product_id = ?`, productID).
		Scan(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

func (repo *ProductRepoImpl) DecrementQuantity(db *gorm.DB, schema string, productID uuid.UUID, qty int) error {
	result := db.Table(schema+".product").
		Where("id = ? AND product_quantity >= ?", productID, qty).
		UpdateColumn("product_quantity", gorm.Expr("product_quantity - ?", qty))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("insufficient stock for product %s", productID)
	}
	// Auto-mark sold out when stock reaches zero
	db.Table(schema+".product").
		Where("id = ? AND product_quantity = 0", productID).
		UpdateColumn("is_out_of_stock", true)
	return nil
}

func (repo *ProductRepoImpl) GetAllByTagIDs(db *gorm.DB, schema string, tagIDs []uuid.UUID, pagination domains.Pagination) ([]domains.Product, int, error) {
	if len(tagIDs) == 0 {
		return repo.GetAll(db, schema, pagination)
	}

	// Build safe IN clause — uuid.UUID.String() produces only hex digits and hyphens
	idLits := make([]string, len(tagIDs))
	for i, id := range tagIDs {
		idLits[i] = fmt.Sprintf("'%s'", id.String())
	}
	inClause := strings.Join(idLits, ",")

	var total int64
	if err := db.Raw(fmt.Sprintf(`
		SELECT COUNT(DISTINCT p.id) FROM %s.product p
		JOIN %s.product_tag_dto ptd ON ptd.product_id = p.id
		WHERE ptd.tag_id IN (%s)`, schema, schema, inClause)).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	var products []domains.Product
	if err := db.Raw(fmt.Sprintf(`
		SELECT DISTINCT p.* FROM %s.product p
		JOIN %s.product_tag_dto ptd ON ptd.product_id = p.id
		WHERE ptd.tag_id IN (%s)
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?`, schema, schema, inClause), pagination.Limit, pagination.Offset()).
		Scan(&products).Error; err != nil {
		return nil, 0, err
	}

	for i, p := range products {
		var images []domains.ProductImage
		db.Raw(`SELECT * FROM `+schema+`.product_image WHERE product_id = ? AND is_active = true`, p.ID).Scan(&images)
		products[i].Images = images
	}

	return products, int(total), nil
}
