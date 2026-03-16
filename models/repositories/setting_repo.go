package repositories

import (
	"backend/models/domains"

	"gorm.io/gorm"
)

type SettingRepo interface {
	Create(db *gorm.DB, schema string, setting domains.Setting) error
	GetByGroupName(db *gorm.DB, schema, groupName string) ([]domains.Setting, error)
	GetByGroupAndSubGroupName(db *gorm.DB, schema, groupName, subGroupName string) ([]domains.Setting, error)
	UpdateBySubGroupName(db *gorm.DB, schema string, group []domains.Setting) error
}
