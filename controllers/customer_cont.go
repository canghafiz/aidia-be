package controllers

import "github.com/gin-gonic/gin"

type CustomerCont interface {
	Create(ctx *gin.Context)
	GetAll(ctx *gin.Context)
	GetByID(ctx *gin.Context)
}
