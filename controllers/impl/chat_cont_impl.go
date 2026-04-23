package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	chatreq "backend/models/requests/chat"
	"backend/models/services"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatContImpl struct {
	ChatServ  services.ChatServ
	GuestRepo repositories.GuestRepo
	UserRepo  repositories.UsersRepo
	Db        *gorm.DB
	JwtKey    string
}

func NewChatContImpl(
	chatServ services.ChatServ,
	guestRepo repositories.GuestRepo,
	userRepo repositories.UsersRepo,
	db *gorm.DB,
	jwtKey string,
) *ChatContImpl {
	return &ChatContImpl{
		ChatServ:  chatServ,
		GuestRepo: guestRepo,
		UserRepo:  userRepo,
		Db:        db,
		JwtKey:    jwtKey,
	}
}

// sseToken reads token from Authorization header or ?token= query param.
// Browsers using EventSource cannot set custom headers, so ?token= is the fallback.
func sseToken(ctx *gin.Context) string {
	if t := helpers.GetJwtToken(ctx); t != "" {
		return t
	}
	return ctx.Query("token")
}

// sseHeaders sets all required SSE + CORS headers.
func sseHeaders(ctx *gin.Context) {
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, Cache-Control")
	ctx.Header("Access-Control-Expose-Headers", "Content-Type")
	ctx.Header("X-Accel-Buffering", "no")
}

// writeSSE writes a named SSE event and flushes. Returns false on write error.
func writeSSE(ctx *gin.Context, eventType string, data interface{}) bool {
	b, _ := json.Marshal(data)
	_, err := fmt.Fprintf(ctx.Writer, "event: %s\ndata: %s\n\n", eventType, b)
	ctx.Writer.Flush()
	return err == nil
}

// GetConversations godoc
// @Summary      List conversations (SSE)
// @Description  SSE stream. Sends event:init with full conversation list on connect, then event:update on any new activity.
// @Description  Auth: Bearer token in Authorization header, OR ?token= query param (for browser EventSource).
// @Tags         Chat
// @Produce      text/event-stream
// @Param        client_id  path   string  true   "Client ID (UUID)"
// @Param        platform   path   string  true   "Platform (telegram|whatsapp)"
// @Param        token      query  string  false  "JWT token (use when EventSource cannot set Authorization header)"
// @Param        page       query  int     false  "Page (default 1)"
// @Param        limit      query  int     false  "Conversations per page (default 50)"
// @Success      200  {string}  string  "SSE stream — event:init | event:update | comment:heartbeat"
// @Failure      401  {string}  string  "event:error"
// @Router       /client/{client_id}/chats/{platform} [get]
func (cont *ChatContImpl) GetConversations(ctx *gin.Context) {
	accessToken := sseToken(ctx)

	clientID, err := uuid.Parse(ctx.Param("client_id"))
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": "invalid client_id"})
		return
	}

	platform := ctx.Param("platform")

	if _, err := helpers.DecodeJWT(accessToken, cont.JwtKey); err != nil {
		ctx.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
		return
	}

	page, limit := 1, 50
	if raw := ctx.Query("page"); raw != "" {
		if n, e := strconv.Atoi(raw); e == nil && n > 0 {
			page = n
		}
	}
	if raw := ctx.Query("limit"); raw != "" {
		if n, e := strconv.Atoi(raw); e == nil && n > 0 {
			limit = n
		}
	}

	sseHeaders(ctx)

	result, err := cont.ChatServ.GetConversations(accessToken, clientID, platform, domains.Pagination{Page: page, Limit: limit})
	if err != nil {
		writeSSE(ctx, "error", gin.H{"message": err.Error()})
		return
	}

	if !writeSSE(ctx, "init", result) {
		return
	}

	// Subscribe to all activity for this tenant
	h := helpers.GetChatHub()
	tenantID := clientID.String()
	ch := h.SubscribeToTenant(tenantID)
	defer h.Unsubscribe(tenantID, "", ch)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	clientGone := ctx.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			log.Printf("[ChatSSE:list] client disconnected tenant=%s", tenantID)
			return
		case <-ticker.C:
			if _, err := fmt.Fprintf(ctx.Writer, ": heartbeat\n\n"); err != nil {
				return
			}
			ctx.Writer.Flush()
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(ctx.Writer, "event: update\ndata: %s\n\n", msg); err != nil {
				return
			}
			ctx.Writer.Flush()
		}
	}
}

// GetConversationDetail godoc
// @Summary      Conversation detail — messages + real-time (SSE)
// @Description  SSE stream. On connect sends event:init with guest info + latest N messages (DESC, newest first).
// @Description  Stays connected and streams event:message for new incoming messages.
// @Description  **Lazy load older messages**: reconnect with ?before_id=<oldest_message_id> to get the previous batch.
// @Description  When before_id is set the connection closes after delivering the batch (no streaming needed).
// @Description  Auth: Bearer token in Authorization header, OR ?token= query param (for browser EventSource).
// @Tags         Chat
// @Produce      text/event-stream
// @Param        client_id  path   string  true   "Client ID (UUID)"
// @Param        platform   path   string  true   "Platform (telegram|whatsapp)"
// @Param        guest_id   path   string  true   "Guest ID (UUID)"
// @Param        token      query  string  false  "JWT token (for browser EventSource)"
// @Param        before_id  query  string  false  "Cursor: fetch messages older than this message ID (lazy load)"
// @Param        limit      query  int     false  "Messages per batch (default 50, max 100)"
// @Success      200  {string}  string  "SSE stream — event:init | event:message | comment:heartbeat"
// @Failure      401  {string}  string  "event:error"
// @Failure      404  {string}  string  "event:error"
// @Router       /client/{client_id}/chats/{platform}/{guest_id} [get]
func (cont *ChatContImpl) GetConversationDetail(ctx *gin.Context) {
	accessToken := sseToken(ctx)

	clientID, err := uuid.Parse(ctx.Param("client_id"))
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": "invalid client_id"})
		return
	}

	platform := ctx.Param("platform")

	guestID, err := uuid.Parse(ctx.Param("guest_id"))
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": "invalid guest_id"})
		return
	}

	if _, err := helpers.DecodeJWT(accessToken, cont.JwtKey); err != nil {
		ctx.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
		return
	}

	var beforeID *uuid.UUID
	if raw := ctx.Query("before_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			ctx.AbortWithStatusJSON(400, gin.H{"error": "invalid before_id"})
			return
		}
		beforeID = &parsed
	}

	limit := 50
	if raw := ctx.Query("limit"); raw != "" {
		if n, e := strconv.Atoi(raw); e == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	sseHeaders(ctx)

	result, err := cont.ChatServ.GetConversationDetail(accessToken, clientID, guestID, platform, beforeID, limit)
	if err != nil {
		writeSSE(ctx, "error", gin.H{"message": err.Error()})
		return
	}

	if !writeSSE(ctx, "init", result) {
		return
	}

	// Cursor/lazy-load request: just deliver the batch and close — no streaming needed.
	if beforeID != nil {
		return
	}

	// Initial load: stay connected and stream new messages for this guest.
	h := helpers.GetChatHub()
	tenantID := clientID.String()
	ch := h.Subscribe(tenantID, guestID.String())
	defer h.Unsubscribe(tenantID, guestID.String(), ch)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	clientGone := ctx.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			log.Printf("[ChatSSE:detail] client disconnected tenant=%s guest=%s", tenantID, guestID)
			return
		case <-ticker.C:
			if _, err := fmt.Fprintf(ctx.Writer, ": heartbeat\n\n"); err != nil {
				return
			}
			ctx.Writer.Flush()
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(ctx.Writer, "event: message\ndata: %s\n\n", msg); err != nil {
				return
			}
			ctx.Writer.Flush()
		}
	}
}

// MarkAsRead godoc
// @Summary      Mark conversation as read
// @Description  Sets is_read=true for the guest conversation
// @Tags         Chat
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true  "Client ID (UUID)"
// @Param        platform   path  string  true  "Platform (telegram|whatsapp)"
// @Param        guest_id   path  string  true  "Guest ID (UUID)"
// @Success      200  {object}  helpers.ApiResponse
// @Failure      401  {object}  helpers.ApiResponse
// @Failure      404  {object}  helpers.ApiResponse
// @Router       /client/{client_id}/chats/{platform}/{guest_id}/read [patch]
func (cont *ChatContImpl) MarkAsRead(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": "invalid client_id"})
		return
	}

	guestID, err := helpers.ParseUUID(ctx, "guest_id")
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": "invalid guest_id"})
		return
	}

	if err := cont.ChatServ.MarkAsRead(accessToken, clientID, guestID); err != nil {
		ctx.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"success": true, "message": "Marked as read"})
}

// SendManualReply godoc
// @Summary      Send manual reply
// @Description  Send a message as the operator (is_human=true, role=assistant)
// @Tags         Chat
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true  "Client ID (UUID)"
// @Param        platform   path  string  true  "Platform (telegram|whatsapp)"
// @Param        guest_id   path  string  true  "Guest ID (UUID)"
// @Param        body       body  chatreq.SendMessageRequest  true  "Request body"
// @Success      200  {object}  helpers.ApiResponse
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      401  {object}  helpers.ApiResponse
// @Router       /client/{client_id}/chats/{platform}/{guest_id}/messages [post]
func (cont *ChatContImpl) SendManualReply(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": "invalid client_id"})
		return
	}

	platform := ctx.Param("platform")

	guestID, err := helpers.ParseUUID(ctx, "guest_id")
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": "invalid guest_id"})
		return
	}

	var req chatreq.SendMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || req.Message == "" {
		ctx.AbortWithStatusJSON(400, gin.H{"error": "message is required"})
		return
	}

	if err := cont.ChatServ.SendManualReply(accessToken, clientID, guestID, req.Message, platform); err != nil {
		if err.Error() == "recipient number is not registered on WhatsApp" {
			ctx.AbortWithStatusJSON(400, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "whatsapp business phone number is connected but not registered yet" {
			ctx.AbortWithStatusJSON(400, gin.H{"error": err.Error()})
			return
		}
		ctx.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"success": true, "message": "Message sent"})
}

func (cont *ChatContImpl) SendTemplateMessage(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": "invalid client_id"})
		return
	}

	platform := ctx.Param("platform")
	if strings.ToLower(strings.TrimSpace(platform)) != "whatsapp" {
		ctx.AbortWithStatusJSON(400, gin.H{"error": "template messages are only supported for whatsapp"})
		return
	}

	guestID, err := helpers.ParseUUID(ctx, "guest_id")
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": "invalid guest_id"})
		return
	}

	var req chatreq.SendTemplateMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TemplateName) == "" {
		ctx.AbortWithStatusJSON(400, gin.H{"error": "template_name is required"})
		return
	}

	if err := cont.ChatServ.SendTemplateMessage(accessToken, clientID, guestID, req.TemplateName, req.LanguageCode, req.BodyParams); err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"success": true, "message": "Template message sent"})
}

// InitTelegramChat godoc
// @Summary      Get Telegram start link for a registered customer
// @Description  Returns a Telegram deep link (t.me/Bot?start=cust_{id}) to share with the customer. Once the customer clicks it and starts the bot, this endpoint is blocked for that customer.
// @Tags         Chat
// @Produce      json
// @Security     BearerAuth
// @Param        client_id    path  string  true  "Client ID (UUID)"
// @Param        customer_id  path  int     true  "Customer ID"
// @Success      200  {object}  helpers.ApiResponse
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      401  {object}  helpers.ApiResponse
// @Router       /client/{client_id}/customers/{customer_id}/telegram/chat [post]
func (cont *ChatContImpl) InitTelegramChat(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": "invalid client_id"})
		return
	}

	customerID, err := strconv.Atoi(ctx.Param("customer_id"))
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": "invalid customer_id"})
		return
	}

	startLink, err := cont.ChatServ.InitTelegramChat(accessToken, clientID, customerID)
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"success": true, "data": gin.H{"start_link": startLink}})
}
