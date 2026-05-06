package impl

import (
	"backend/helpers"
	"context"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WhatsAppWhatsmeowContImpl struct {
	Hub    *helpers.WhatsmeowHub
	Db     *gorm.DB
	JwtKey string
}

func NewWhatsAppWhatsmeowContImpl(hub *helpers.WhatsmeowHub, db *gorm.DB, jwtKey string) *WhatsAppWhatsmeowContImpl {
	return &WhatsAppWhatsmeowContImpl{Hub: hub, Db: db, JwtKey: jwtKey}
}

func (cont *WhatsAppWhatsmeowContImpl) getSchema(ctx *gin.Context) string {
	header := ctx.GetHeader("Authorization")
	tokenStr := strings.TrimPrefix(header, "Bearer ")
	claims, err := helpers.DecodeJWT(tokenStr, cont.JwtKey)
	if err != nil {
		return ""
	}
	schema, _ := claims["tenant_schema"].(string)
	return schema
}

// GetQRStream godoc
// @Summary      Stream WhatsApp QR Code (whatsmeow)
// @Description  SSE endpoint that streams QR codes for pairing a personal WhatsApp number. Connect with EventSource in the browser. Events: "qr" (scan the code), "connected" (success), "timeout", "error".
// @Tags         WhatsApp Whatsmeow
// @Produce      text/event-stream
// @Security     BearerAuth
// @Param        client_id  path  string  true  "Client ID"
// @Success      200
// @Router       /client/{client_id}/whatsapp/whatsmeow/qr [get]
func (cont *WhatsAppWhatsmeowContImpl) GetQRStream(ctx *gin.Context) {
	if cont.Hub == nil {
		ctx.JSON(503, gin.H{"success": false, "data": "whatsmeow not available"})
		return
	}

	schema := cont.getSchema(ctx)
	if schema == "" {
		ctx.JSON(401, gin.H{"success": false, "data": "unauthorized"})
		return
	}

	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("X-Accel-Buffering", "no")

	reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Minute)
	defer cancel()

	qrChan, err := cont.Hub.ConnectWithQR(reqCtx, schema)
	if err != nil {
		ctx.SSEvent("error", gin.H{"error": err.Error()})
		ctx.Writer.Flush()
		return
	}

	ctx.Stream(func(w io.Writer) bool {
		select {
		case item, ok := <-qrChan:
			if !ok {
				return false
			}
			switch item.Event {
			case "code":
				ctx.SSEvent("qr", gin.H{"code": item.Code})
				return true
			case "success":
				ctx.SSEvent("connected", gin.H{"status": "connected", "phone": cont.Hub.GetPhone(schema)})
				return false
			case "timeout":
				ctx.SSEvent("timeout", gin.H{"error": "QR code expired, please try again"})
				return false
			default:
				if item.Error != nil {
					ctx.SSEvent("error", gin.H{"error": item.Error.Error()})
				}
				return false
			}
		case <-reqCtx.Done():
			return false
		}
	})
}

// GetStatus godoc
// @Summary      WhatsApp Whatsmeow Connection Status
// @Description  Returns whether the tenant's personal WhatsApp (whatsmeow) is connected.
// @Tags         WhatsApp Whatsmeow
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true  "Client ID"
// @Success      200  {object}  helpers.ApiResponse
// @Router       /client/{client_id}/whatsapp/whatsmeow/status [get]
func (cont *WhatsAppWhatsmeowContImpl) GetStatus(ctx *gin.Context) {
	if cont.Hub == nil {
		ctx.JSON(200, gin.H{"success": true, "data": gin.H{"connected": false, "phone": ""}})
		return
	}

	schema := cont.getSchema(ctx)
	if schema == "" {
		ctx.JSON(401, gin.H{"success": false, "data": "unauthorized"})
		return
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"connected": cont.Hub.IsConnected(schema),
			"phone":     cont.Hub.GetPhone(schema),
		},
	})
}

// Disconnect godoc
// @Summary      Disconnect WhatsApp Whatsmeow
// @Description  Logs out and removes the personal WhatsApp (whatsmeow) session for the tenant.
// @Tags         WhatsApp Whatsmeow
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true  "Client ID"
// @Success      200  {object}  helpers.ApiResponse
// @Router       /client/{client_id}/whatsapp/whatsmeow/disconnect [delete]
func (cont *WhatsAppWhatsmeowContImpl) Disconnect(ctx *gin.Context) {
	if cont.Hub == nil {
		ctx.JSON(503, gin.H{"success": false, "data": "whatsmeow not available"})
		return
	}

	schema := cont.getSchema(ctx)
	if schema == "" {
		ctx.JSON(401, gin.H{"success": false, "data": "unauthorized"})
		return
	}

	if err := cont.Hub.Disconnect(schema); err != nil {
		ctx.JSON(500, gin.H{"success": false, "data": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"success": true, "data": "WhatsApp (whatsmeow) disconnected"})
}
