package impl

import (
	"backend/exceptions"
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	"backend/models/services"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChatContImpl struct {
	ChatServ   services.ChatServ
	GuestRepo  repositories.GuestRepo
	UserRepo   repositories.UsersRepo
	Db         *gorm.DB
	JwtKey     string
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

// GetConversations godoc
// @Summary      Get all conversations
// @Description  Get list of all conversations with pagination
// @Tags         Chat
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string  true  "Client ID"
// @Param        page       query     int     false "Page"
// @Param        limit      query     int     false "Limit"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/chats [get]
func (cont *ChatContImpl) GetConversations(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	// Get pagination
	page := ctx.DefaultQuery("page", "1")
	limit := ctx.DefaultQuery("limit", "20")

	pagination := domains.Pagination{}
	fmt.Sscanf(page, "%d", &pagination.Page)
	fmt.Sscanf(limit, "%d", &pagination.Limit)

	result, err := cont.ChatServ.GetConversations(accessToken, clientID, pagination)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: result}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}

// GetConversationDetail godoc
// @Summary      Get conversation detail
// @Description  Get detail of a conversation with messages
// @Tags         Chat
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true  "Client ID"
// @Param        guest_id   path  string  true  "Guest ID"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/chats/{guest_id} [get]
func (cont *ChatContImpl) GetConversationDetail(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	guestID, err := helpers.ParseUUID(ctx, "guest_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, err := cont.ChatServ.GetConversationDetail(accessToken, clientID, guestID)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: result}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}

// Stream godoc
// @Summary      Chat real-time stream (SSE)
// @Description  Subscribe to real-time chat updates via Server-Sent Events
// @Tags         Chat
// @Produce      text/event-stream
// @Security     BearerAuth
// @Param        client_id  path  string  true  "Client ID"
// @Router       /client/{client_id}/chats/stream [get]
func (cont *ChatContImpl) Stream(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	// Validate token
	if _, err := helpers.DecodeJWT(accessToken, cont.JwtKey); err != nil {
		exceptions.ErrorHandler(ctx, fmt.Errorf("invalid token"))
		return
	}

	// Get tenant schema from user
	user, err := cont.UserRepo.GetByUserId(cont.Db, clientID)
	if err != nil {
		exceptions.ErrorHandler(ctx, fmt.Errorf("user not found"))
		return
	}

	if user.TenantSchema == nil || *user.TenantSchema == "" {
		exceptions.ErrorHandler(ctx, fmt.Errorf("tenant schema not found"))
		return
	}

	schema := helpers.NormalizeSchema(*user.TenantSchema)
	tenantID := clientID.String()

	// Set SSE headers
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("X-Accel-Buffering", "no")

	// Subscribe to hub (all guests in tenant)
	h := helpers.GetChatHub()
	ch := h.SubscribeToTenant(tenantID)
	defer func() {
		h.Unsubscribe(tenantID, "", ch)
	}()

	// Send initial event
	initEvent := map[string]interface{}{
		"event": "connected",
		"data": map[string]interface{}{
			"message":     "Connected to chat stream",
			"tenant_id":   tenantID,
			"schema":      schema,
			"timestamp":   time.Now().Format(time.RFC3339),
		},
	}
	initData, _ := json.Marshal(initEvent)
	if _, err := fmt.Fprintf(ctx.Writer, "data: %s\n\n", initData); err != nil {
		log.Printf("[ChatSSE] write init error: %v", err)
		return
	}
	ctx.Writer.Flush()

	// Listen for updates or disconnect
	clientGone := ctx.Request.Context().Done()
	for {
		select {
		case <-clientGone:
			log.Printf("[ChatSSE] client disconnected, tenant: %s", tenantID)
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(ctx.Writer, "data: %s\n\n", msg); err != nil {
				log.Printf("[ChatSSE] write error: %v", err)
				return
			}
			ctx.Writer.Flush()
		}
	}
}

// SendManualReply godoc
// @Summary      Send manual reply to guest
// @Description  Send manual reply to a guest (for manual mode)
// @Tags         Chat
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true  "Client ID"
// @Param        guest_id   path  string  true  "Guest ID"
// @Param        request    body  object  true  "Request body with message field"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/chats/{guest_id}/messages [post]
func (cont *ChatContImpl) SendManualReply(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	guestID, err := helpers.ParseUUID(ctx, "guest_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	var request struct {
		Message string `json:"message" validate:"required"`
	}

	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	if request.Message == "" {
		exceptions.ErrorHandler(ctx, fmt.Errorf("message is required"))
		return
	}

	err = cont.ChatServ.SendManualReply(accessToken, clientID, guestID, request.Message)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: map[string]string{"message": "Message sent"}}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}
