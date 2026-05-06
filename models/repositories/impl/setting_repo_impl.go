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
		if err := db.Exec(`
			INSERT INTO `+domains.SettingTable(schema)+` (group_name, sub_group_name, name, value)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (sub_group_name, name) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
		`, setting.GroupName, setting.SubgroupName, setting.Name, setting.Value).Error; err != nil {
			return err
		}
	}
	return nil
}

func (repo *SettingRepoImpl) GetByName(db *gorm.DB, schema, groupName, subGroupName, name string) (*domains.Setting, error) {
	var s domains.Setting
	err := db.Table(domains.SettingTable(schema)).
		Where("group_name = ? AND sub_group_name = ? AND name = ?", groupName, subGroupName, name).
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (repo *SettingRepoImpl) UpsertByName(db *gorm.DB, schema, groupName, subGroupName, name, value string) error {
	return db.Exec(`
		INSERT INTO `+domains.SettingTable(schema)+` (group_name, sub_group_name, name, value)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (sub_group_name, name) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
	`, groupName, subGroupName, name, value).Error
}

func (repo *SettingRepoImpl) DeleteByName(db *gorm.DB, schema, groupName, subGroupName, name string) error {
	return db.Table(domains.SettingTable(schema)).
		Where("group_name = ? AND sub_group_name = ? AND name = ?", groupName, subGroupName, name).
		Delete(&domains.Setting{}).Error
}
