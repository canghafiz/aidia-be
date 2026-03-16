package domains

import (
	"time"

	"github.com/google/uuid"
)

type Setting struct {
	ID           uuid.UUID `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	GroupName    string    `gorm:"column:group_name;not null"`
	SubgroupName string    `gorm:"column:sub_group_name;not null"`
	Name         string    `gorm:"column:name;not null;uniqueIndex:uq_setting_name"`
	Value        string    `gorm:"column:value"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func SettingTable(schema string) string {
	if schema == "" || schema == "public" {
		return "public.setting"
	}
	return schema + ".setting"
}
