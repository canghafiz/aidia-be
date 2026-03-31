package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	"backend/models/services"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TelegramContImpl struct {
	GuestRepo        repositories.GuestRepo
	GuestMessageRepo repositories.GuestMessageRepo
	SettingRepo      repositories.SettingRepo
	UserRepo         repositories.UsersRepo
	N8NServ          services.N8NServ
	Db               *gorm.DB
}

func NewTelegramContImpl(
	guestRepo repositories.GuestRepo,
	guestMessageRepo repositories.GuestMessageRepo,
	settingRepo repositories.SettingRepo,
	userRepo repositories.UsersRepo,
	n8nServ services.N8NServ,
	db *gorm.DB,
) *TelegramContImpl {
	return &TelegramContImpl{
		GuestRepo:        guestRepo,
		GuestMessageRepo: guestMessageRepo,
		SettingRepo:      settingRepo,
		UserRepo:         userRepo,
		N8NServ:          n8nServ,
		Db:               db,
	}
}

// TelegramWebhookRequest represents incoming Telegram webhook payload
type TelegramWebhookRequest struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		MessageID int    `json:"message_id"`
		From      *User  `json:"from"`
		Chat      *Chat  `json:"chat"`
		Date      int64  `json:"date"`
		Text      string `json:"text"`
		Contact   *Contact `json:"contact"`
	} `json:"message"`
}

type User struct {
	ID        int    `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type Chat struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Username string `json:"username"`
}

type Contact struct {
	PhoneNumber string `json:"phone_number"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	UserID      int    `json:"user_id"`
}

// Webhook godoc
// @Summary      Telegram Webhook
// @Description  Receive incoming Telegram messages
// @Tags         Telegram
// @Accept       json
// @Produce      json
// @Param        schema  path  string  true  "Tenant Schema"
// @Param        request body  TelegramWebhookRequest  true  "Telegram Webhook Payload"
// @Success      200     {object}  helpers.ApiResponse
// @Failure      400     {object}  helpers.ApiResponse
// @Failure      500     {object}  helpers.ApiResponse
// @Router       /api/v1/webhook/telegram/{schema} [post]
func (cont *TelegramContImpl) Webhook(ctx *gin.Context) {
	schema := ctx.Param("schema")
	if schema == "" {
		log.Printf("[Telegram Webhook] schema required")
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	var payload TelegramWebhookRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		log.Printf("[Telegram Webhook] bind error: %v", err)
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	// Ignore if not a message or is from bot
	if payload.Message == nil || payload.Message.From.IsBot {
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	// Extract data
	chatID := fmt.Sprintf("%d", payload.Message.Chat.ID)
	userID := fmt.Sprintf("%d", payload.Message.From.ID)
	
	// Handle contact share (phone number)
	if payload.Message.Contact != nil {
		log.Printf("[Telegram Webhook] Contact received: %s, %s", 
			payload.Message.Contact.PhoneNumber, 
			payload.Message.Contact.FirstName)
		
		// Update guest with phone number
		guest, err := cont.GuestRepo.FindByTelegramChatID(cont.Db, schema, chatID)
		if err == nil && guest != nil {
			guest.Phone = payload.Message.Contact.PhoneNumber
			cont.GuestRepo.Update(cont.Db, schema, *guest)
			log.Printf("[Telegram Webhook] Guest phone updated: %s", guest.Phone)
		}
		
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}
	
	text := ""
	if payload.Message.Text != "" {
		text = payload.Message.Text
	}
	messageID := payload.Message.MessageID

	log.Printf("[Telegram Webhook] schema: %s, chat_id: %s, user: %s, message: %s",
		schema, chatID, userID, text)

	// Get tenant info from schema (preload Tenant)
	user, err := cont.UserRepo.FindByUsernameOrEmail(cont.Db, schema, "Tenant")
	if err != nil || user == nil {
		log.Printf("[Telegram Webhook] tenant not found: %v", err)
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	// Check if tenant exists
	if user.Tenant == nil || user.Tenant.TenantID == uuid.Nil {
		log.Printf("[Telegram Webhook] tenant data not found for schema: %s", schema)
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	tenantID := user.Tenant.TenantID

	// Get Telegram bot token from setting
	setting, err := cont.SettingRepo.GetByGroupAndSubGroupName(cont.Db, schema, "integration", "Telegram")
	if err != nil {
		log.Printf("[Telegram Webhook] failed to get setting: %v", err)
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	botToken := ""
	for _, s := range setting {
		if s.Name == "telegram-bot-token" {
			botToken = s.Value
			break
		}
	}

	if botToken == "" {
		log.Printf("[Telegram Webhook] bot token not configured")
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	// Find or create guest
	guest, err := cont.GuestRepo.FindByTelegramChatID(cont.Db, schema, chatID)
	if err != nil {
		// Build name from first_name + last_name
		fullName := payload.Message.From.FirstName
		if payload.Message.From.LastName != "" {
			fullName += " " + payload.Message.From.LastName
		}

		// Create new guest with ALL available data
		guest = &domains.Guest{
			TenantID:         &tenantID,
			Identity:         chatID,              // Set identity = chat_id
			Username:         payload.Message.From.Username, // Telegram username (@xxx)
			Phone:            "",                  // Empty (Telegram tidak kasih phone)
			Name:             fullName,            // Full name
			TelegramChatID:   chatID,
			TelegramUsername: payload.Message.From.Username,
			Sosmed: domains.JSONB{
				"id":            float64(payload.Message.From.ID),
				"first_name":    payload.Message.From.FirstName,
				"last_name":     payload.Message.From.LastName,
				"username":      payload.Message.From.Username,
				"is_bot":        payload.Message.From.IsBot,
			},
			IsActive:   true,
			IsRead:     false,
			IsTakeOver: false,
		}

		if err := cont.GuestRepo.Create(cont.Db, schema, *guest); err != nil {
			log.Printf("[Telegram Webhook] failed to create guest: %v", err)
			ctx.JSON(200, gin.H{"status": "ok"})
			return
		}

		// Reload guest with ID
		guest, _ = cont.GuestRepo.FindByTelegramChatID(cont.Db, schema, chatID)
		
		// AUTO: Send phone request to new guest asynchronously
		go cont.sendPhoneRequest(schema, chatID, botToken)
	} else {
		// Update existing guest data if changed
		needsUpdate := false
		
		// Update username if changed
		if payload.Message.From.Username != "" && guest.TelegramUsername != payload.Message.From.Username {
			guest.TelegramUsername = payload.Message.From.Username
			guest.Username = payload.Message.From.Username
			needsUpdate = true
		}
		
		// Update name if changed
		fullName := payload.Message.From.FirstName
		if payload.Message.From.LastName != "" {
			fullName += " " + payload.Message.From.LastName
		}
		if guest.Name != fullName {
			guest.Name = fullName
			needsUpdate = true
		}
		
		// Update sosmed JSON
		guest.Sosmed = domains.JSONB{
			"id":            float64(payload.Message.From.ID),
			"first_name":    payload.Message.From.FirstName,
			"last_name":     payload.Message.From.LastName,
			"username":      payload.Message.From.Username,
			"is_bot":        payload.Message.From.IsBot,
		}
		needsUpdate = true
		
		if needsUpdate {
			cont.GuestRepo.Update(cont.Db, schema, *guest)
		}
	}

	// Save message to database
	newMessage := domains.GuestMessage{
		GuestID:           guest.ID,
		Role:              "user",
		Type:              "text",
		Message:           text,
		TelegramMessageID: &messageID,
		Platform:          "telegram",
		IsActive:          true,
	}

	if err := cont.GuestMessageRepo.Create(cont.Db, schema, newMessage); err != nil {
		log.Printf("[Telegram Webhook] failed to save message: %v", err)
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	// Update guest last_message_at
	now := time.Now()
	guest.LastMessageAt = &now
	cont.GuestRepo.Update(cont.Db, schema, *guest)

	// Broadcast to SSE hub
	eventData := map[string]interface{}{
		"event": "new_message",
		"data": map[string]interface{}{
			"guest_id":     guest.ID.String(),
			"guest_name":   guest.Name,
			"telegram_id":  guest.TelegramChatID,
			"message":      text,
			"role":         "user",
			"timestamp":    now.Format(time.RFC3339),
			"platform":     "telegram",
		},
	}

	eventJSON, _ := json.Marshal(eventData)
	h := helpers.GetChatHub()
	h.BroadcastToGuest(tenantID.String(), guest.ID.String(), string(eventJSON))

	// Get message history for context
	history, _ := cont.GuestMessageRepo.GetLatestMessages(cont.Db, schema, guest.ID, 20)

	// Get custom prompt from setting
	customPrompt := ""
	settingPrompt, err := cont.SettingRepo.GetByGroupAndSubGroupName(cont.Db, schema, "integration", "Telegram Bot")
	if err == nil && len(settingPrompt) > 0 {
		for _, s := range settingPrompt {
			if s.Name == "ai-prompt" {
				customPrompt = s.Value
				break
			}
		}
	}
	
	log.Printf("[DEBUG] customPrompt from DB: %s", customPrompt)

	// Forward to n8n for AI processing
	go func() {
		log.Printf("[n8n] forwarding to n8n: schema=%s, guest_id=%s, prompt=%s", schema, guest.ID.String(), customPrompt)
		
		n8nResp, err := cont.N8NServ.ProcessMessage(schema, guest.ID.String(), chatID, text, customPrompt, history)
		if err != nil {
			log.Printf("[n8n] error: %v", err)
			return
		}

		// Save AI reply to database
		aiMessage := domains.GuestMessage{
			GuestID:  guest.ID,
			Role:     "assistant",
			Type:     "text",
			Message:  n8nResp.Reply,
			IsHuman:  false,
			IsActive: true,
		}
		cont.GuestMessageRepo.Create(cont.Db, schema, aiMessage)

		// Send reply to Telegram
		tgClient := helpers.NewTelegramClient(botToken)
		_, err = tgClient.SendMessage(chatID, n8nResp.Reply)
		if err != nil {
			log.Printf("[Telegram] send message error: %v", err)
			return
		}

		// Broadcast AI reply to SSE hub
		replyEvent := map[string]interface{}{
			"event": "new_message",
			"data": map[string]interface{}{
				"guest_id":     guest.ID.String(),
				"guest_name":   guest.Name,
				"telegram_id":  guest.TelegramChatID,
				"message":      n8nResp.Reply,
				"role":         "assistant",
				"is_human":     false,
				"timestamp":    time.Now().Format(time.RFC3339),
				"platform":     "telegram",
			},
		}
		replyJSON, _ := json.Marshal(replyEvent)
		h.BroadcastToGuest(tenantID.String(), guest.ID.String(), string(replyJSON))

		log.Printf("[Telegram Webhook] AI reply sent to chat_id: %s", chatID)
	}()

	ctx.JSON(200, gin.H{"status": "ok"})
}

// GetAIPromptForSchema godoc
// @Summary      Get AI Prompt for Schema (Internal API for n8n)
// @Description  Get custom AI prompt for specific tenant schema (internal API for n8n)
// @Tags         Telegram
// @Produce      json
// @Param        schema  path  string  true  "Tenant Schema"
// @Success      200    {object}  helpers.ApiResponse{data=telegram.AIPromptResponse}
// @Failure      400    {object}  helpers.ApiResponse
// @Failure      500    {object}  helpers.ApiResponse
// @Router       /api/v1/internal/telegram/{schema}/ai-prompt [get]
func (cont *TelegramContImpl) GetAIPromptForSchema(ctx *gin.Context) {
	schema := ctx.Param("schema")
	if schema == "" {
		ctx.JSON(400, gin.H{"error": "schema required"})
		return
	}

	// Get prompt from setting
	settingPrompt, err := cont.SettingRepo.GetByGroupAndSubGroupName(cont.Db, schema, "integration", "Telegram Bot")
	if err != nil {
		ctx.JSON(500, gin.H{"error": "failed to get prompt"})
		return
	}

	prompt := "Anda adalah asisten AI untuk restoran ini. Tugas Anda:\n1. Bantu customer lihat menu/produk\n2. Bantu customer buat pesanan\n3. Jawab pertanyaan seputar restoran\n4. Selalu konfirmasi sebelum membuat pesanan\n\nBalas dengan bahasa yang ramah dan natural."

	for _, s := range settingPrompt {
		if s.Name == "ai-prompt" {
			prompt = s.Value
			break
		}
	}

	ctx.JSON(200, gin.H{
		"prompt": prompt,
		"schema": schema,
	})
}

// RequestPhone godoc
// @Summary      Request Phone Number from User
// @Description  Send keyboard with "Share Phone Number" button to user
// @Tags         Telegram
// @Produce      json
// @Param        client_id  path  string  true  "Client ID"
// @Param        request    body  telegram.RequestPhoneRequest  true  "Request Phone Request"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Security     BearerAuth
// @Router       /client/{client_id}/telegram/request-phone [post]
func (cont *TelegramContImpl) RequestPhone(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var request struct {
		ChatID string `json:"chat_id" validate:"required"`
	}

	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Get schema
	schema, err := helpers.GetSchema(cont.Db, cont.UserRepo, clientID)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Get bot token
	setting, err := cont.SettingRepo.GetByGroupAndSubGroupName(cont.Db, schema, "integration", "Telegram")
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	botToken := ""
	for _, s := range setting {
		if s.Name == "telegram-bot-token" {
			botToken = s.Value
			break
		}
	}

	if botToken == "" {
		ctx.JSON(500, gin.H{"error": "bot token not configured"})
		return
	}

	// Send message with contact button
	tgClient := helpers.NewTelegramClient(botToken)
	
	// Create custom keyboard with contact button
	keyboard := map[string]interface{}{
		"keyboard": [][]map[string]interface{}{
			{
				{
					"text":            "📱 Share Phone Number",
					"request_contact": true,
				},
			},
		},
		"resize_keyboard":   true,
		"one_time_keyboard": true,
	}

	message := "Hello! To complete your registration, please share your phone number with us.\n\nClick the button below to share:"

	err = tgClient.SendMessageWithKeyboard(request.ChatID, message, keyboard)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data: map[string]string{
			"message": "Phone request sent successfully",
		},
	}

	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
}

var _ interface{} = (*TelegramContImpl)(nil)
