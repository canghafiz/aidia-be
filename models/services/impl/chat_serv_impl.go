package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	"backend/models/services"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatServImpl struct {
	Db             *gorm.DB
	JwtKey         string
	GuestRepo      repositories.GuestRepo
	GuestMessageRepo repositories.GuestMessageRepo
	UserRepo       repositories.UsersRepo
}

func NewChatServImpl(
	db *gorm.DB,
	jwtKey string,
	guestRepo repositories.GuestRepo,
	guestMessageRepo repositories.GuestMessageRepo,
	userRepo repositories.UsersRepo,
) *ChatServImpl {
	return &ChatServImpl{
		Db:               db,
		JwtKey:           jwtKey,
		GuestRepo:        guestRepo,
		GuestMessageRepo: guestMessageRepo,
		UserRepo:         userRepo,
	}
}

func (serv *ChatServImpl) GetConversations(accessToken string, clientID uuid.UUID, pagination domains.Pagination) (interface{}, error) {
	// Validate token
	_, err := helpers.DecodeJWT(accessToken, serv.JwtKey)
	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	// Get tenant schema from user
	user, err := serv.UserRepo.GetByUserId(serv.Db, clientID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	if user.TenantSchema == nil {
		return nil, fmt.Errorf("tenant schema not found")
	}

	// Get conversations (guests) with their latest message
	var guests []domains.Guest
	var total int64

	tx := serv.Db.Table("guest").Where("tenant_id = ?", clientID)
	tx.Count(&total)

	err = tx.Offset((pagination.Page - 1) * pagination.Limit).
		Limit(pagination.Limit).
		Order("last_message_at DESC").
		Find(&guests).Error
	if err != nil {
		return nil, err
	}

	// Build response
	type ConversationResponse struct {
		GuestID       uuid.UUID  `json:"guest_id"`
		GuestName     string     `json:"guest_name"`
		TelegramID    string     `json:"telegram_id"`
		LastMessage   string     `json:"last_message"`
		LastMessageAt *time.Time `json:"last_message_at"`
		IsRead        bool       `json:"is_read"`
		IsTakeOver    bool       `json:"is_take_over"`
	}

	var responses []ConversationResponse
	for _, guest := range guests {
		// Get latest message
		var latestMsg domains.GuestMessage
		serv.Db.Table("guest_message").
			Where("guest_id = ?", guest.ID).
			Order("created_at DESC").
			First(&latestMsg)

		lastMsg := ""
		if latestMsg.ID != uuid.Nil {
			lastMsg = latestMsg.Message
		}

		responses = append(responses, ConversationResponse{
			GuestID:       guest.ID,
			GuestName:     guest.Name,
			TelegramID:    guest.TelegramChatID,
			LastMessage:   lastMsg,
			LastMessageAt: guest.LastMessageAt,
			IsRead:        guest.IsRead,
			IsTakeOver:    guest.IsTakeOver,
		})
	}

	return map[string]interface{}{
		"conversations": responses,
		"total":         total,
		"page":          pagination.Page,
		"limit":         pagination.Limit,
	}, nil
}

func (serv *ChatServImpl) GetConversationDetail(accessToken string, clientID, guestID uuid.UUID) (interface{}, error) {
	// Validate token
	_, err := helpers.DecodeJWT(accessToken, serv.JwtKey)
	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	// Get schema
	schema, err := helpers.GetSchema(serv.Db, serv.UserRepo, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	// Get guest
	guest, err := serv.GuestRepo.FindByID(serv.Db, schema, guestID)
	if err != nil {
		return nil, fmt.Errorf("guest not found")
	}

	// Validate tenant access
	if guest.TenantID == nil || *guest.TenantID != clientID {
		return nil, fmt.Errorf("access denied")
	}

	// Get messages
	messages, err := serv.GuestMessageRepo.FindByGuestID(serv.Db, schema, guestID, 100)
	if err != nil {
		return nil, err
	}

	type MessageResponse struct {
		ID        uuid.UUID `json:"id"`
		Role      string    `json:"role"`
		Message   string    `json:"message"`
		Type      string    `json:"type"`
		CreatedAt time.Time `json:"created_at"`
	}

	var msgResponses []MessageResponse
	for _, msg := range messages {
		msgResponses = append(msgResponses, MessageResponse{
			ID:        msg.ID,
			Role:      msg.Role,
			Message:   msg.Message,
			Type:      msg.Type,
			CreatedAt: msg.CreatedAt,
		})
	}

	return map[string]interface{}{
		"guest": map[string]interface{}{
			"guest_id":       guest.ID,
			"guest_name":     guest.Name,
			"telegram_id":    guest.TelegramChatID,
			"telegram_user":  guest.TelegramUsername,
			"is_take_over":   guest.IsTakeOver,
			"is_read":        guest.IsRead,
			"conversation_state": guest.ConversationState,
		},
		"messages": msgResponses,
	}, nil
}

func (serv *ChatServImpl) SendManualReply(accessToken string, clientID, guestID uuid.UUID, message string) error {
	// Validate token
	if _, err := helpers.DecodeJWT(accessToken, serv.JwtKey); err != nil {
		return fmt.Errorf("invalid token")
	}

	// Get schema
	schema, err := helpers.GetSchema(serv.Db, serv.UserRepo, clientID)
	if err != nil {
		return fmt.Errorf("failed to get schema: %w", err)
	}

	// Get guest
	guest, err := serv.GuestRepo.FindByID(serv.Db, schema, guestID)
	if err != nil {
		return fmt.Errorf("guest not found")
	}

	// Validate tenant access
	if guest.TenantID == nil || *guest.TenantID != clientID {
		return fmt.Errorf("access denied")
	}

	// Create message
	newMsg := domains.GuestMessage{
		GuestID: guestID,
		Role:    "assistant",
		Type:    "text",
		Message: message,
		IsHuman: true,
		IsActive: true,
	}

	err = serv.GuestMessageRepo.Create(serv.Db, schema, newMsg)
	if err != nil {
		return err
	}

	// Update guest last_message_at
	now := time.Now()
	guest.LastMessageAt = &now
	guest.IsRead = false
	serv.GuestRepo.Update(serv.Db, schema, *guest)

	// Broadcast to SSE hub
	eventData := map[string]interface{}{
		"event": "new_message",
		"data": map[string]interface{}{
			"guest_id":   guestID.String(),
			"guest_name": guest.Name,
			"message":    message,
			"role":       "assistant",
			"is_human":   true,
			"timestamp":  now.Format(time.RFC3339),
		},
	}

	eventJSON, _ := json.Marshal(eventData)
	hub := helpers.GetChatHub()
	hub.BroadcastToGuest(clientID.String(), guestID.String(), string(eventJSON))

	// TODO: Send to Telegram API here (will be implemented separately)

	return nil
}

var _ services.ChatServ = (*ChatServImpl)(nil)
