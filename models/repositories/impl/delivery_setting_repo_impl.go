package impl

import (
	"backend/models/domains"

	"gorm.io/gorm"
)

type DeliverySettingRepoImpl struct{}

func NewDeliverySettingRepoImpl() *DeliverySettingRepoImpl {
	return &DeliverySettingRepoImpl{}
}

func (repo *DeliverySettingRepoImpl) Create(db *gorm.DB, schema string, settings []domains.Setting) error {
	return db.Table(schema + ".setting").Create(&settings).Error
}

func (repo *DeliverySettingRepoImpl) Update(db *gorm.DB, schema string, settings []domains.Setting) error {
	for _, s := range settings {
		if err := db.Table(schema+".setting").
			Where("sub_group_name = ? AND name = ?", s.SubgroupName, s.Name).
			Updates(map[string]interface{}{
				"value": s.Value,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (repo *DeliverySettingRepoImpl) GetAll(db *gorm.DB, schema string) ([]domains.Setting, error) {
	var settings []domains.Setting
	if err := db.Table(schema+".setting").
		Where("group_name = ?", "delivery").
		Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (repo *DeliverySettingRepoImpl) GetBySubGroupName(db *gorm.DB, schema string, subGroupName string) ([]domains.Setting, error) {
	var settings []domains.Setting
	if err := db.Table(schema+".setting").
		Where("group_name = ? AND sub_group_name = ?", "delivery", subGroupName).
		Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (repo *DeliverySettingRepoImpl) Delete(db *gorm.DB, schema string, subGroupName string) error {
	return db.Table(schema+".setting").
		Where("group_name = ? AND sub_group_name = ?", "delivery", subGroupName).
		Delete(&domains.Setting{}).Error
}
