package controllers

import "github.com/gin-gonic/gin"

type WhatsAppOAuthCont interface {
	Connect(ctx *gin.Context)
	Status(ctx *gin.Context)
	Disconnect(ctx *gin.Context)
}
