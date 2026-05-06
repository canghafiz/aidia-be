package controllers

import "github.com/gin-gonic/gin"

type WhatsAppCont interface {
	VerifyWebhook(ctx *gin.Context)
	Webhook(ctx *gin.Context)
	GetAIContextForSchema(ctx *gin.Context)
	// Global webhook — single URL for all tenants, routed via phone_number_id
	VerifyWebhookGlobal(ctx *gin.Context)
	WebhookGlobal(ctx *gin.Context)
	// Internal endpoints called by n8n during WhatsApp registration flow
	UpdateWhatsAppGuestConversationState(ctx *gin.Context)
	CompleteWhatsAppGuestRegistration(ctx *gin.Context)
}
