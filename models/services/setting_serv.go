package services

import (
	"backend/models/repositories"
	req "backend/models/requests/setting"
	"backend/models/responses/setting"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SettingServ interface {
	GetNotification(accessToken string) (*setting.GroupResponse, error)
	GetIntegration(accessToken string) (*setting.GroupResponse, error)
	UpdateBySubgroupName(accessToken, subGroupName string, requests req.UpdateBySubgroupRequest) error
	UpdateTelegramBotToken(accessToken string, clientID uuid.UUID, botToken string) error
	GetJwtKey() string
	GetDb() *gorm.DB
	GetUserRepo() repositories.UsersRepo
	GetByGroupAndSubGroupName(db *gorm.DB, schema, group, subGroup string) ([]interface{}, error)
	UpdateBySubGroupNameForSchema(db *gorm.DB, schema, subGroupName, name, value string) error
	GetAIPrompts(schema string) (map[string]string, error)
	UpdateAIPromptSection(accessToken, schema, section, prompt string) error
}
