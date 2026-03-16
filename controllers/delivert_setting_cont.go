package controllers

import "github.com/gin-gonic/gin"

type DeliverySettingCont interface {
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	GetAll(ctx *gin.Context)
	GetBySubGroupName(ctx *gin.Context)
	Delete(ctx *gin.Context)
}
