package impl

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductTagRepoImpl struct{}

func NewProductTagRepoImpl() *ProductTagRepoImpl {
	return &ProductTagRepoImpl{}
}

func (r *ProductTagRepoImpl) Create(db *gorm.DB, schema string, tag domains.ProductTag) error {
	return db.Table(schema + ".product_tag").Create(&tag).Error
}

func (r *ProductTagRepoImpl) Update(db *gorm.DB, schema string, tag domains.ProductTag) error {
	return db.Table(schema+".product_tag").Where("id = ?", tag.ID).
		Updates(map[string]interface{}{"name": tag.Name}).Error
}

func (r *ProductTagRepoImpl) GetAll(db *gorm.DB, schema string) ([]domains.ProductTag, error) {
	var tags []domains.ProductTag
	err := db.Table(schema + ".product_tag").Order("name ASC").Find(&tags).Error
	return tags, err
}

func (r *ProductTagRepoImpl) GetByID(db *gorm.DB, schema string, id uuid.UUID) (*domains.ProductTag, error) {
	var tag domains.ProductTag
	if err := db.Table(schema+".product_tag").Where("id = ?", id).First(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *ProductTagRepoImpl) GetByName(db *gorm.DB, schema string, name string) (*domains.ProductTag, error) {
	var tag domains.ProductTag
	if err := db.Table(schema+".product_tag").Where("name = ?", name).First(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *ProductTagRepoImpl) Delete(db *gorm.DB, schema string, id uuid.UUID) error {
	return db.Table(schema+".product_tag").Where("id = ?", id).Delete(&domains.ProductTag{}).Error
}

func (r *ProductTagRepoImpl) GetTagsByProductID(db *gorm.DB, schema string, productID uuid.UUID) ([]domains.ProductTag, error) {
	var tags []domains.ProductTag
	err := db.Raw(`
		SELECT pt.* FROM `+schema+`.product_tag pt
		JOIN `+schema+`.product_tag_dto ptd ON ptd.tag_id = pt.id
		WHERE ptd.product_id = ?`, productID).Scan(&tags).Error
	return tags, err
}

func (r *ProductTagRepoImpl) CreateTagDtos(db *gorm.DB, schema string, dtos []domains.ProductTagDto) error {
	return db.Table(schema + ".product_tag_dto").Create(&dtos).Error
}

func (r *ProductTagRepoImpl) DeleteTagDtosByProductID(db *gorm.DB, schema string, productID uuid.UUID) error {
	return db.Table(schema+".product_tag_dto").Where("product_id = ?", productID).
		Delete(&domains.ProductTagDto{}).Error
}
