package controllers

import "github.com/gin-gonic/gin"

type WhatsAppWhatsmeowCont interface {
	// GetQRStream streams QR codes as SSE events until the phone is paired ("connected") or times out.
	GetQRStream(ctx *gin.Context)
	// GetStatus returns the whatsmeow connection status for the client's tenant.
	GetStatus(ctx *gin.Context)
	// Disconnect logs out and removes the whatsmeow session for the client's tenant.
	Disconnect(ctx *gin.Context)
}
