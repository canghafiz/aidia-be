package controllers

import "github.com/gin-gonic/gin"

type TelegramCont interface {
	Webhook(ctx *gin.Context)
}
