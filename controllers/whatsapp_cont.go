package controllers

import "github.com/gin-gonic/gin"

type WhatsAppCont interface {
	VerifyWebhook(ctx *gin.Context)
	Webhook(ctx *gin.Context)
	GetAIContextForSchema(ctx *gin.Context)
	// Global webhook — satu URL untuk semua tenant, routing via phone_number_id
	VerifyWebhookGlobal(ctx *gin.Context)
	WebhookGlobal(ctx *gin.Context)
}
