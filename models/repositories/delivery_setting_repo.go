package repositories

import (
	"backend/models/domains"

	"gorm.io/gorm"
)

type DeliverySettingRepo interface {
	Create(db *gorm.DB, schema string, settings []domains.Setting) error
	Update(db *gorm.DB, schema string, settings []domains.Setting) error
	GetAll(db *gorm.DB, schema string) ([]domains.Setting, error)
	GetBySubGroupName(db *gorm.DB, schema string, subGroupName string) ([]domains.Setting, error)
	Delete(db *gorm.DB, schema string, subGroupName string) error
}
