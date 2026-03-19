package controllers

import "github.com/gin-gonic/gin"

type OrderCont interface {
	Create(ctx *gin.Context)
	GetAll(ctx *gin.Context)
	GetByID(ctx *gin.Context)
	UpdateStatus(ctx *gin.Context)
}
