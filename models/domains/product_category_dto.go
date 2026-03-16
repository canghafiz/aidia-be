package domains

import "github.com/google/uuid"

type ProductCategoryDto struct {
	ProductID  uuid.UUID `gorm:"column:product_id;primaryKey"`
	CategoryID uuid.UUID `gorm:"column:category_id;primaryKey"`
}
