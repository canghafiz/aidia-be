package controllers

import "github.com/gin-gonic/gin"

type ApprovalCont interface {
	Approval(context *gin.Context)
	GetAll(context *gin.Context)
	Delete(context *gin.Context)
}
