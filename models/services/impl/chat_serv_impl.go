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

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatServImpl struct {
	Db               *gorm.DB
	JwtKey           string
	GuestRepo        repositories.GuestRepo
	GuestMessageRepo repositories.GuestMessageRepo
	UserRepo         repositories.UsersRepo
	SettingRepo      repositories.SettingRepo
}

func NewChatServImpl(
	db *gorm.DB,
	jwtKey string,
	guestRepo repositories.GuestRepo,
	guestMessageRepo repositories.GuestMessageRepo,
	userRepo repositories.UsersRepo,
	settingRepo repositories.SettingRepo,
) *ChatServImpl {
	return &ChatServImpl{
		Db:               db,
		JwtKey:           jwtKey,
		GuestRepo:        guestRepo,
		GuestMessageRepo: guestMessageRepo,
		UserRepo:         userRepo,
		SettingRepo:      settingRepo,
	}
}

// resolveTenant validates token and returns (schema, tenantID, error).
// guest.tenant_id = public.tenant.tenant_id, NOT the user_id (clientID).
func (serv *ChatServImpl) resolveTenant(accessToken string, clientID uuid.UUID) (schema string, tenantID uuid.UUID, err error) {
	if _, err = helpers.DecodeJWT(accessToken, serv.JwtKey); err != nil {
		return "", uuid.Nil, fmt.Errorf("invalid token")
	}

	user, err := serv.UserRepo.GetByUserId(serv.Db, clientID)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("user not found")
	}

	if user.TenantSchema == nil || *user.TenantSchema == "" {
		return "", uuid.Nil, fmt.Errorf("tenant schema not found")
	}

	if user.Tenant == nil || user.Tenant.TenantID == uuid.Nil {
		return "", uuid.Nil, fmt.Errorf("tenant not found")
	}

	return *user.TenantSchema, user.Tenant.TenantID, nil
}

func (serv *ChatServImpl) GetConversations(accessToken string, clientID uuid.UUID, pagination domains.Pagination) (interface{}, error) {
	schema, tenantID, err := serv.resolveTenant(accessToken, clientID)
	if err != nil {
		return nil, err
	}

	guests, total, err := serv.GuestRepo.FindAllByTenantID(serv.Db, schema, tenantID, pagination)
	if err != nil {
		return nil, err
	}

	type ConversationItem struct {
		GuestID          uuid.UUID  `json:"guest_id"`
		Name             string     `json:"name"`
		Identity         string     `json:"identity"`
		Phone            string     `json:"phone"`
		Username         string     `json:"username"`
		PlatformChatID   string     `json:"platform_chat_id"`
		PlatformUsername string     `json:"platform_username"`
		LastMessage      string     `json:"last_message"`
		LastMessageAt    *time.Time `json:"last_message_at"`
		IsRead           bool       `json:"is_read"`
		IsTakeOver       bool       `json:"is_take_over"`
		IsActive         bool       `json:"is_active"`
	}

	items := make([]ConversationItem, 0, len(guests))
	for _, g := range guests {
		msgs, _ := serv.GuestMessageRepo.FindByGuestIDCursor(serv.Db, schema, g.ID, nil, 1)
		lastMsg := ""
		if len(msgs) > 0 {
			lastMsg = msgs[0].Message
		}

		items = append(items, ConversationItem{
			GuestID:          g.ID,
			Name:             g.Name,
			Identity:         g.Identity,
			Phone:            g.Phone,
			Username:         g.Username,
			PlatformChatID:   g.PlatformChatID,
			PlatformUsername: g.PlatformUsername,
			LastMessage:      lastMsg,
			LastMessageAt:    g.LastMessageAt,
			IsRead:           g.IsRead,
			IsTakeOver:       g.IsTakeOver,
			IsActive:         g.IsActive,
		})
	}

	return map[string]interface{}{
		"conversations": items,
		"total":         total,
		"page":          pagination.Page,
		"limit":         pagination.Limit,
	}, nil
}

func (serv *ChatServImpl) GetConversationDetail(accessToken string, clientID, guestID uuid.UUID, beforeID *uuid.UUID, limit int) (interface{}, error) {
	schema, _, err := serv.resolveTenant(accessToken, clientID)
	if err != nil {
		return nil, err
	}

	// Schema is already scoped to the tenant — if guest exists here, access is valid
	guest, err := serv.GuestRepo.FindByID(serv.Db, schema, guestID)
	if err != nil {
		return nil, fmt.Errorf("guest not found")
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	messages, err := serv.GuestMessageRepo.FindByGuestIDCursor(serv.Db, schema, guestID, beforeID, limit)
	if err != nil {
		return nil, err
	}

	type MessageItem struct {
		ID        uuid.UUID `json:"id"`
		Role      string    `json:"role"`
		Message   string    `json:"message"`
		Type      string    `json:"type"`
		IsHuman   bool      `json:"is_human"`
		Platform  string    `json:"platform"`
		CreatedAt time.Time `json:"created_at"`
	}

	// DB returns DESC (newest first); reverse to ASC (oldest first) for chat display.
	// next_cursor always points to the oldest message in the original DESC slice (last element).
	var nextCursor *string
	if len(messages) == limit {
		oldest := messages[len(messages)-1].ID.String()
		nextCursor = &oldest
	}

	msgItems := make([]MessageItem, len(messages))
	for i, m := range messages {
		// fill in reverse order so index 0 = oldest
		msgItems[len(messages)-1-i] = MessageItem{
			ID:        m.ID,
			Role:      m.Role,
			Message:   m.Message,
			Type:      m.Type,
			IsHuman:   m.IsHuman,
			Platform:  m.Platform,
			CreatedAt: m.CreatedAt,
		}
	}

	return map[string]interface{}{
		"guest": map[string]interface{}{
			"guest_id":          guest.ID,
			"name":              guest.Name,
			"identity":          guest.Identity,
			"phone":             guest.Phone,
			"username":          guest.Username,
			"platform_chat_id":  guest.PlatformChatID,
			"platform_username": guest.PlatformUsername,
			"is_take_over":      guest.IsTakeOver,
			"is_read":           guest.IsRead,
			"is_active":         guest.IsActive,
			"last_message_at":   guest.LastMessageAt,
		},
		"messages":    msgItems,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != nil,
	}, nil
}

func (serv *ChatServImpl) MarkAsRead(accessToken string, clientID, guestID uuid.UUID) error {
	schema, _, err := serv.resolveTenant(accessToken, clientID)
	if err != nil {
		return err
	}

	if _, err := serv.GuestRepo.FindByID(serv.Db, schema, guestID); err != nil {
		return fmt.Errorf("guest not found")
	}

	return serv.GuestRepo.MarkAsRead(serv.Db, schema, guestID)
}

func (serv *ChatServImpl) SendManualReply(accessToken string, clientID, guestID uuid.UUID, message string) error {
	schema, _, err := serv.resolveTenant(accessToken, clientID)
	if err != nil {
		return err
	}

	guest, err := serv.GuestRepo.FindByID(serv.Db, schema, guestID)
	if err != nil {
		return fmt.Errorf("guest not found")
	}

	newMsg := domains.GuestMessage{
		GuestID:  guestID,
		Role:     "assistant",
		Type:     "text",
		Message:  message,
		IsHuman:  true,
		IsActive: true,
	}

	if err := serv.GuestMessageRepo.Create(serv.Db, schema, newMsg); err != nil {
		return err
	}

	now := time.Now()
	guest.LastMessageAt = &now
	guest.IsRead = false
	serv.GuestRepo.Update(serv.Db, schema, *guest)

	// Send to Telegram so the guest actually receives the reply
	if guest.PlatformChatID != "" {
		settings, err := serv.SettingRepo.GetByGroupAndSubGroupName(serv.Db, schema, "integration", "Telegram")
		if err == nil {
			botToken := ""
			for _, s := range settings {
				if s.Name == "telegram-bot-token" {
					botToken = s.Value
					break
				}
			}
			if botToken != "" {
				tgClient := helpers.NewTelegramClient(botToken)
				if _, err := tgClient.SendMessage(guest.PlatformChatID, message); err != nil {
					log.Printf("[ChatServ] SendManualReply Telegram send error guest=%s: %v", guestID, err)
				}
			}
		}
	}

	// Broadcast to dashboard SSE
	eventData := map[string]interface{}{
		"event": "new_message",
		"data": map[string]interface{}{
			"guest_id":   guestID.String(),
			"guest_name": guest.Name,
			"message":    message,
			"role":       "assistant",
			"is_human":   true,
			"created_at": now.Format(time.RFC3339),
		},
	}
	eventJSON, _ := json.Marshal(eventData)
	h := helpers.GetChatHub()
	h.BroadcastToGuest(clientID.String(), guestID.String(), string(eventJSON))
	h.BroadcastToTenant(clientID.String(), string(eventJSON))

	return nil
}

var _ services.ChatServ = (*ChatServImpl)(nil)
