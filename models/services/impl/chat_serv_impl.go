package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	"backend/models/services"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
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
	CustomerRepo     repositories.CustomerRepo
}

func NewChatServImpl(
	db *gorm.DB,
	jwtKey string,
	guestRepo repositories.GuestRepo,
	guestMessageRepo repositories.GuestMessageRepo,
	userRepo repositories.UsersRepo,
	settingRepo repositories.SettingRepo,
	customerRepo repositories.CustomerRepo,
) *ChatServImpl {
	return &ChatServImpl{
		Db:               db,
		JwtKey:           jwtKey,
		GuestRepo:        guestRepo,
		GuestMessageRepo: guestMessageRepo,
		UserRepo:         userRepo,
		SettingRepo:      settingRepo,
		CustomerRepo:     customerRepo,
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

func (serv *ChatServImpl) getTelegramBotToken(schema string) (string, error) {
	settings, err := serv.SettingRepo.GetByGroupAndSubGroupName(serv.Db, schema, "integration", "Telegram")
	if err != nil || len(settings) == 0 {
		return "", fmt.Errorf("telegram integration is not configured for this account")
	}
	for _, s := range settings {
		if s.Name == "telegram-bot-token" && s.Value != "" {
			return s.Value, nil
		}
	}
	return "", fmt.Errorf("telegram bot token is not configured")
}

func (serv *ChatServImpl) getWhatsAppClient(schema string) (*helpers.WhatsAppClient, error) {
	settings, err := serv.SettingRepo.GetByGroupAndSubGroupName(serv.Db, schema, "integration", "WhatsApp")
	if err != nil || len(settings) == 0 {
		return nil, fmt.Errorf("whatsapp integration is not configured for this account")
	}

	phoneNumberID := ""
	accessToken := ""
	for _, s := range settings {
		switch s.Name {
		case "whatsapp-phone-number-id":
			phoneNumberID = s.Value
		case "whatsapp-access-token":
			accessToken = s.Value
		}
	}

	if phoneNumberID == "" || accessToken == "" {
		return nil, fmt.Errorf("whatsapp integration is incomplete for this account")
	}

	return helpers.NewWhatsAppClient(phoneNumberID, accessToken), nil
}

func (serv *ChatServImpl) getWhatsAppRegistrationPin(schema string) string {
	settings, err := serv.SettingRepo.GetByGroupAndSubGroupName(serv.Db, schema, "integration", "WhatsApp")
	if err == nil {
		for _, s := range settings {
			if s.Name == "whatsapp-registration-pin" && strings.TrimSpace(s.Value) != "" {
				return strings.TrimSpace(s.Value)
			}
		}
	}

	envKeys := []string{
		"WHATSAPP_REGISTRATION_PIN",
		"META_WHATSAPP_REGISTRATION_PIN",
		"WHATSAPP_PHONE_PIN",
	}
	for _, key := range envKeys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}

	return ""
}

func (serv *ChatServImpl) ensureWhatsAppRegistrationPin(schema string) string {
	if pin := serv.getWhatsAppRegistrationPin(schema); pin != "" {
		return pin
	}

	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return ""
	}
	pin := strconv.FormatInt(n.Int64()+100000, 10)
	err = serv.Db.Exec(
		`INSERT INTO `+schema+`.setting (id, group_name, sub_group_name, name, value, created_at, updated_at)
		VALUES (gen_random_uuid(), 'integration', 'WhatsApp', 'whatsapp-registration-pin', ?, NOW(), NOW())
		ON CONFLICT (sub_group_name, name) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
		pin,
	).Error
	if err != nil {
		log.Printf("[ChatServ] failed to persist whatsapp registration pin schema=%s: %v", schema, err)
		return ""
	}
	return pin
}

func (serv *ChatServImpl) GetConversations(accessToken string, clientID uuid.UUID, platform string, pagination domains.Pagination) (interface{}, error) {
	schema, tenantID, err := serv.resolveTenant(accessToken, clientID)
	if err != nil {
		return nil, err
	}

	if platform == "telegram" {
		if _, err := serv.getTelegramBotToken(schema); err != nil {
			return nil, err
		}
	}

	guests, total, err := serv.GuestRepo.FindAllByTenantID(serv.Db, schema, tenantID, platform, pagination)
	if err != nil {
		return nil, err
	}

	type ConversationItem struct {
		GuestID          uuid.UUID  `json:"guest_id"`
		Name             string     `json:"name"`
		Identity         string     `json:"identity"`
		Phone            string     `json:"phone"`
		Username         string     `json:"username"`
		Platform         string     `json:"platform"`
		PlatformChatID   string     `json:"platform_chat_id"`
		LastMessage      string     `json:"last_message"`
		LastMessageAt    *time.Time `json:"last_message_at"`
		IsRead           bool       `json:"is_read"`
		IsTakeOver       bool       `json:"is_take_over"`
		IsActive         bool       `json:"is_active"`
	}

	items := make([]ConversationItem, 0, len(guests))
	for _, g := range guests {
		msgs, _ := serv.GuestMessageRepo.FindByGuestIDCursor(serv.Db, schema, g.ID, platform, nil, 1)
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
			Platform:         pickGuestPlatform(g, platform),
			PlatformChatID:   g.PlatformChatID,
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

func pickGuestPlatform(g domains.Guest, fallback string) string {
	return strings.ToLower(strings.TrimSpace(g.Platform))
}

func (serv *ChatServImpl) GetConversationDetail(accessToken string, clientID, guestID uuid.UUID, platform string, beforeID *uuid.UUID, limit int) (interface{}, error) {
	schema, _, err := serv.resolveTenant(accessToken, clientID)
	if err != nil {
		return nil, err
	}

	if platform == "telegram" {
		if _, err := serv.getTelegramBotToken(schema); err != nil {
			return nil, err
		}
	}

	guest, err := serv.GuestRepo.FindByID(serv.Db, schema, guestID)
	if err != nil {
		return nil, fmt.Errorf("guest not found")
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	messages, err := serv.GuestMessageRepo.FindByGuestIDCursor(serv.Db, schema, guestID, platform, beforeID, limit)
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
	var nextCursor *string
	if len(messages) == limit {
		oldest := messages[len(messages)-1].ID.String()
		nextCursor = &oldest
	}

	msgItems := make([]MessageItem, len(messages))
	for i, m := range messages {
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
			"platform":          pickGuestPlatform(*guest, platform),
			"platform_chat_id":  guest.PlatformChatID,
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

func (serv *ChatServImpl) SendManualReply(accessToken string, clientID, guestID uuid.UUID, message, platform string) error {
	schema, _, err := serv.resolveTenant(accessToken, clientID)
	if err != nil {
		return err
	}

	if platform == "telegram" {
		if _, err := serv.getTelegramBotToken(schema); err != nil {
			return err
		}
	}

	guest, err := serv.GuestRepo.FindByID(serv.Db, schema, guestID)
	if err != nil {
		return fmt.Errorf("guest not found")
	}

	if platform == "whatsapp" {
		waClient, err := serv.getWhatsAppClient(schema)
		if err != nil {
			return err
		}

		chatID := guest.PlatformChatID
		if chatID == "" {
			chatID = guest.Identity
		}
		if chatID == "" {
			return fmt.Errorf("whatsapp guest chat id is missing")
		}

		if err := waClient.SendMessage(chatID, message); err != nil {
			log.Printf("[ChatServ] SendManualReply WhatsApp send error guest=%s: %v", guestID, err)
			if helpers.IsWhatsAppRecipientNotRegistered(err) {
				return fmt.Errorf("recipient number is not registered on WhatsApp")
			}
			if helpers.IsWhatsAppBusinessNotRegistered(err) {
				pin := serv.ensureWhatsAppRegistrationPin(schema)
				if pin == "" {
					return fmt.Errorf("whatsapp business phone number is connected but not registered yet")
				}
				log.Printf("[ChatServ] attempting WhatsApp phone registration for schema=%s phone_number_id=%s", schema, waClient.PhoneNumberID)
				if regErr := waClient.RegisterPhoneNumber(pin); regErr != nil {
					log.Printf("[ChatServ] WhatsApp registration failed schema=%s: %v", schema, regErr)
					return fmt.Errorf("whatsapp business phone number is connected but not registered yet")
				}
				if retryErr := waClient.SendMessage(chatID, message); retryErr != nil {
					log.Printf("[ChatServ] SendManualReply WhatsApp retry send error guest=%s: %v", guestID, retryErr)
					if helpers.IsWhatsAppRecipientNotRegistered(retryErr) {
						return fmt.Errorf("recipient number is not registered on WhatsApp")
					}
					if helpers.IsWhatsAppBusinessNotRegistered(retryErr) {
						return fmt.Errorf("whatsapp business phone number is connected but not registered yet")
					}
					return fmt.Errorf("failed to send WhatsApp message: %w", retryErr)
				}
			} else {
				return fmt.Errorf("failed to send WhatsApp message: %w", err)
			}
		}
	}

	newMsg := domains.GuestMessage{
		GuestID:  guestID,
		Role:     "assistant",
		Type:     "text",
		Message:  message,
		IsHuman:  true,
		IsActive: true,
		Platform: platform,
	}

	if err := serv.GuestMessageRepo.Create(serv.Db, schema, newMsg); err != nil {
		return err
	}

	now := time.Now()
	guest.LastMessageAt = &now
	guest.IsRead = false
	serv.GuestRepo.Update(serv.Db, schema, *guest)

	if platform == "telegram" && guest.PlatformChatID != "" {
		botToken, err := serv.getTelegramBotToken(schema)
		if err == nil {
			tgClient := helpers.NewTelegramClient(botToken)
			if _, err := tgClient.SendMessage(guest.PlatformChatID, message); err != nil {
				log.Printf("[ChatServ] SendManualReply Telegram send error guest=%s: %v", guestID, err)
			}
		}
	}

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

func (serv *ChatServImpl) SendTemplateMessage(accessToken string, clientID, guestID uuid.UUID, templateName, languageCode string, bodyParams []string) error {
	schema, _, err := serv.resolveTenant(accessToken, clientID)
	if err != nil {
		return err
	}

	guest, err := serv.GuestRepo.FindByID(serv.Db, schema, guestID)
	if err != nil {
		return fmt.Errorf("guest not found")
	}

	waClient, err := serv.getWhatsAppClient(schema)
	if err != nil {
		return err
	}

	chatID := guest.PlatformChatID
	if chatID == "" {
		chatID = guest.Identity
	}
	if chatID == "" {
		return fmt.Errorf("whatsapp guest chat id is missing")
	}

	if err := waClient.SendTemplateMessage(chatID, templateName, languageCode, bodyParams); err != nil {
		log.Printf("[ChatServ] SendTemplateMessage WhatsApp send error guest=%s: %v", guestID, err)
		return err
	}

	templateLabel := strings.TrimSpace(templateName)
	if templateLabel == "" {
		templateLabel = "template"
	}
	preview := fmt.Sprintf("[Template] %s", templateLabel)
	if len(bodyParams) > 0 {
		preview += " - " + strings.Join(bodyParams, ", ")
	}

	newMsg := domains.GuestMessage{
		GuestID:  guestID,
		Role:     "assistant",
		Type:     "template",
		Message:  preview,
		IsHuman:  true,
		IsActive: true,
		Platform: "whatsapp",
	}

	if err := serv.GuestMessageRepo.Create(serv.Db, schema, newMsg); err != nil {
		return err
	}

	now := time.Now()
	guest.LastMessageAt = &now
	guest.IsRead = false
	serv.GuestRepo.Update(serv.Db, schema, *guest)

	eventData := map[string]interface{}{
		"event": "new_message",
		"data": map[string]interface{}{
			"guest_id":   guestID.String(),
			"guest_name": guest.Name,
			"message":    preview,
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

func (serv *ChatServImpl) InitTelegramChat(accessToken string, clientID uuid.UUID, customerID int) (string, error) {
	schema, _, err := serv.resolveTenant(accessToken, clientID)
	if err != nil {
		return "", err
	}

	botToken, err := serv.getTelegramBotToken(schema)
	if err != nil {
		return "", err
	}

	customer, err := serv.CustomerRepo.GetByID(serv.Db, schema, customerID)
	if err != nil {
		return "", fmt.Errorf("customer not found")
	}

	if customer.AccountType != "Telegram" {
		return "", fmt.Errorf("customer is not a Telegram account")
	}

	if customer.Username == nil || *customer.Username == "" {
		return "", fmt.Errorf("customer has no telegram username")
	}

	// Block if customer has already started the bot
	guest, err := serv.GuestRepo.FindByUsername(serv.Db, schema, *customer.Username)
	if err == nil && guest != nil && guest.PlatformChatID != "" {
		return "", fmt.Errorf("customer has already started the bot, use the regular chat interface")
	}

	// Get bot username via getMe
	tgClient := helpers.NewTelegramClient(botToken)
	me, err := tgClient.GetMe()
	if err != nil || me == nil {
		return "", fmt.Errorf("failed to get bot info")
	}

	startLink := fmt.Sprintf("https://t.me/%s?start=cust_%d", me.Result.Username, customerID)
	return startLink, nil
}

var _ services.ChatServ = (*ChatServImpl)(nil)
