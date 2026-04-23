package controllers

import "github.com/gin-gonic/gin"

type PaymentCont interface {
	CreatePlatformCheckout(ctx *gin.Context)
	CreatePaymentFromExisting(ctx *gin.Context)
	GetPlatformInvoices(ctx *gin.Context)
	GetPlatformInvoiceByID(ctx *gin.Context)
	GetAvailableGateways(ctx *gin.Context)
	HandlePlatformWebhookStripe(ctx *gin.Context)
	HandlePlatformWebhookHitPay(ctx *gin.Context)

	CreateClientCheckout(ctx *gin.Context)
	GetClientInvoices(ctx *gin.Context)
	HandleClientWebhookStripe(ctx *gin.Context)
	HandleClientWebhookHitPay(ctx *gin.Context)
}
