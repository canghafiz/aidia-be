package controllers

import "github.com/gin-gonic/gin"

type SettingCont interface {
	GetNotification(context *gin.Context)
	GetIntegration(context *gin.Context)
	UpdateBySubgroupName(context *gin.Context)
	UpdateTelegramBotToken(context *gin.Context)
	GetClientIntegration(context *gin.Context)
	UpdateClientIntegration(context *gin.Context)
	GetClientAIPrompts(context *gin.Context)
	GetClientAIPromptSection(context *gin.Context)
	UpdateClientAIPromptSection(context *gin.Context)
}
