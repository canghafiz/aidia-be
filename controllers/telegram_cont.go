package controllers

import "github.com/gin-gonic/gin"

type TelegramCont interface {
	Webhook(ctx *gin.Context)
	GetAIContextForSchema(ctx *gin.Context)
	GetPublicOrderDetail(ctx *gin.Context)
	UpdateGuestConversationState(ctx *gin.Context)
	CompleteGuestRegistration(ctx *gin.Context)
}
