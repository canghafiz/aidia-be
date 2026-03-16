package controllers

import "github.com/gin-gonic/gin"

type PaymentCont interface {
	CreatePlatformCheckout(ctx *gin.Context)
	CreatePaymentFromExisting(ctx *gin.Context)
	GetPlatformInvoices(ctx *gin.Context)
	GetPlatformInvoiceByID(ctx *gin.Context)
	HandlePlatformWebhook(ctx *gin.Context)

	CreateClientCheckout(ctx *gin.Context)
	GetClientInvoices(ctx *gin.Context)
	HandleClientWebhook(ctx *gin.Context)
}
