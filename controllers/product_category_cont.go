package controllers

import "github.com/gin-gonic/gin"

type ProductCategoryCont interface {
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	GetAll(ctx *gin.Context)
	GetByID(ctx *gin.Context)
	Delete(ctx *gin.Context)
}
