package services

import (
	req "backend/models/requests/setting"
	"backend/models/responses/setting"

	"github.com/google/uuid"
)

type SettingServ interface {
	GetNotification(accessToken string) (*setting.GroupResponse, error)
	GetIntegration(accessToken string) (*setting.GroupResponse, error)
	UpdateBySubgroupName(accessToken, subGroupName string, requests req.UpdateBySubgroupRequest) error
	UpdateTelegramBotToken(accessToken string, clientID uuid.UUID, botToken string) error
}
