package controllers

import "github.com/gin-gonic/gin"

type SettingCont interface {
	GetNotification(context *gin.Context)
	GetIntegration(context *gin.Context)
	UpdateBySubgroupName(context *gin.Context)
}
