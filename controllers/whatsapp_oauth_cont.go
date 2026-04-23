package controllers

import "github.com/gin-gonic/gin"

type WhatsAppOAuthCont interface {
	Connect(ctx *gin.Context)
	Status(ctx *gin.Context)
	Disconnect(ctx *gin.Context)
	GetConfig(ctx *gin.Context) // return META_APP_ID + META_EMBEDDED_SIGNUP_CONFIG_ID untuk FB SDK
}
