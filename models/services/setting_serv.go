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
	GetTelegramAIPrompt(accessToken, schema string) (string, error)
	UpdateTelegramAIPrompt(accessToken, prompt string) error
	UpdateTelegramAIPromptForSchema(accessToken, schema, prompt string) error
	GetJwtKey() string
	GetDb() *gorm.DB
	GetUserRepo() repositories.UsersRepo
}
