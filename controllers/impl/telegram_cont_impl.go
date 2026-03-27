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
	text := payload.Message.Text
	messageID := payload.Message.MessageID

	log.Printf("[Telegram Webhook] schema: %s, chat_id: %s, user: %s, message: %s",
		schema, chatID, userID, text)

	// Get tenant info from schema
	user, err := cont.UserRepo.FindByUsernameOrEmail(cont.Db, schema)
	if err != nil {
		log.Printf("[Telegram Webhook] tenant not found: %v", err)
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
		// Create new guest
		guest = &domains.Guest{
			TenantID:         &tenantID,
			Identity:         chatID,
			TelegramChatID:   chatID,
			TelegramUsername: payload.Message.From.Username,
			Name:             payload.Message.From.FirstName,
			Sosmed: domains.JSONB{
				"id":         float64(payload.Message.From.ID),
				"first_name": payload.Message.From.FirstName,
				"last_name":  payload.Message.From.LastName,
				"username":   payload.Message.From.Username,
				"is_bot":     payload.Message.From.IsBot,
			},
			IsActive: true,
			IsRead:   false,
		}

		if err := cont.GuestRepo.Create(cont.Db, *guest); err != nil {
			log.Printf("[Telegram Webhook] failed to create guest: %v", err)
			ctx.JSON(200, gin.H{"status": "ok"})
			return
		}

		// Reload guest with ID
		guest, _ = cont.GuestRepo.FindByTelegramChatID(cont.Db, schema, chatID)
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

	if err := cont.GuestMessageRepo.Create(cont.Db, newMessage); err != nil {
		log.Printf("[Telegram Webhook] failed to save message: %v", err)
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	// Update guest last_message_at
	now := time.Now()
	guest.LastMessageAt = &now
	cont.GuestRepo.Update(cont.Db, *guest)

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
	history, _ := cont.GuestMessageRepo.GetLatestMessages(cont.Db, guest.ID, 20)

	// Forward to n8n for AI processing
	go func() {
		n8nResp, err := cont.N8NServ.ProcessMessage(schema, guest.ID.String(), chatID, text, history)
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
		cont.GuestMessageRepo.Create(cont.Db, aiMessage)

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

var _ interface{} = (*TelegramContImpl)(nil)
