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
	GetByName(db *gorm.DB, schema, groupName, subGroupName, name string) (*domains.Setting, error)
	UpsertByName(db *gorm.DB, schema, groupName, subGroupName, name, value string) error
	DeleteByName(db *gorm.DB, schema, groupName, subGroupName, name string) error
}
