package controllers

import "github.com/gin-gonic/gin"

type TelegramCont interface {
	Webhook(ctx *gin.Context)
	GetAIPromptForSchema(ctx *gin.Context)
	RequestPhone(ctx *gin.Context)
}
