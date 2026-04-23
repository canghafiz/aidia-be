package controllers

import "github.com/gin-gonic/gin"

type ChatCont interface {
	GetConversations(ctx *gin.Context)
	GetConversationDetail(ctx *gin.Context)
	MarkAsRead(ctx *gin.Context)
	SendManualReply(ctx *gin.Context)
	SendTemplateMessage(ctx *gin.Context)
	InitTelegramChat(ctx *gin.Context)
}
