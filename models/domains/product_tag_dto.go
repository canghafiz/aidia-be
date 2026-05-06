package domains

import "github.com/google/uuid"

type ProductTagDto struct {
	ProductID uuid.UUID `gorm:"column:product_id;not null"`
	TagID     uuid.UUID `gorm:"column:tag_id;not null"`
}
