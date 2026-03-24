package controllers

import "github.com/gin-gonic/gin"

type OrderPaymentCont interface {
	GetAll(ctx *gin.Context)
	GetByID(ctx *gin.Context)
	UpdateStatus(ctx *gin.Context)
}
