package controllers

import "github.com/gin-gonic/gin"

type ChatCont interface {
	GetConversations(ctx *gin.Context)
	GetConversationDetail(ctx *gin.Context)
	Stream(ctx *gin.Context)
	SendManualReply(ctx *gin.Context)
}
