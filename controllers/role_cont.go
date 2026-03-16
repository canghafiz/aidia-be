package controllers

import "github.com/gin-gonic/gin"

type RoleCont interface {
	GetRoles(context *gin.Context)
}
