package controllers

import "github.com/gin-gonic/gin"

type SubsCont interface {
	GetCurrentSubs(ctx *gin.Context)
	GetTokenUsage(ctx *gin.Context)
}
