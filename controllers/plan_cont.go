package controllers

import "github.com/gin-gonic/gin"

type PlanCont interface {
	Create(context *gin.Context)
	ToggleIsActive(context *gin.Context)
	Update(context *gin.Context)
	GetById(context *gin.Context)
	GetAll(context *gin.Context)
	Delete(context *gin.Context)
}
