package controllers

import "github.com/gin-gonic/gin"

type KitchenOrderCont interface {
	GetDisplay(ctx *gin.Context)
	Stream(ctx *gin.Context)
	UpdateStatus(ctx *gin.Context)
}
