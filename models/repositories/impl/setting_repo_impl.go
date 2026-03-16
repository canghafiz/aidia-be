package impl

import (
	"backend/models/domains"

	"gorm.io/gorm"
)

type SettingRepoImpl struct {
}

func NewSettingRepoImpl() *SettingRepoImpl {
	return &SettingRepoImpl{}
}

func (repo *SettingRepoImpl) Create(db *gorm.DB, schema string, setting domains.Setting) error {
	return db.Table(schema + ".setting").Create(&setting).Error
}

func (repo *SettingRepoImpl) GetByGroupName(db *gorm.DB, schema, groupName string) ([]domains.Setting, error) {
	var settings []domains.Setting
	if err := db.Table(domains.SettingTable(schema)).
		Where("group_name = ?", groupName).
		Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (repo *SettingRepoImpl) GetByGroupAndSubGroupName(db *gorm.DB, schema, groupName, subGroupName string) ([]domains.Setting, error) {
	var settings []domains.Setting
	if err := db.Table(domains.SettingTable(schema)).
		Where("group_name = ? AND sub_group_name = ?", groupName, subGroupName).
		Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (repo *SettingRepoImpl) UpdateBySubGroupName(db *gorm.DB, schema string, group []domains.Setting) error {
	for _, setting := range group {
		if err := db.Table(domains.SettingTable(schema)).
			Where("sub_group_name = ? AND name = ?", setting.SubgroupName, setting.Name).
			Update("value", setting.Value).Error; err != nil {
			return err
		}
	}
	return nil
}
