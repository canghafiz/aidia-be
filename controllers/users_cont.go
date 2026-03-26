package controllers

import "github.com/gin-gonic/gin"

type UsersCont interface {
	Login(context *gin.Context)
	ChangePw(context *gin.Context)
	Me(context *gin.Context)
	CheckSuperAdminExist(context *gin.Context)
	CreateSuperAdmin(context *gin.Context)
	CreateUser(context *gin.Context)
	UpdateProfileClient(context *gin.Context)
	UpdateProfileNonClient(context *gin.Context)
	EditUserData(context *gin.Context)
	GetByUserId(context *gin.Context)
	GetUsers(context *gin.Context)
	GetClients(context *gin.Context)
	FilterUsers(context *gin.Context)
	DeleteByUserId(context *gin.Context)
}
