package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	"backend/models/services"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WhatsAppContImpl struct {
	GuestRepo              repositories.GuestRepo
	GuestMessageRepo       repositories.GuestMessageRepo
	SettingRepo            repositories.SettingRepo
	UserRepo               repositories.UsersRepo
	ProductRepo            repositories.ProductRepo
	OrderRepo              repositories.OrderRepo
	OrderPaymentRepo       repositories.OrderPaymentRepo
	CustomerRepo           repositories.CustomerRepo
	TenantUsageRepo        repositories.TenantUsageRepo
	N8NServ                services.N8NServ
	WhatsAppConnectionRepo repositories.WhatsAppConnectionRepo
	Db                     *gorm.DB
}

func NewWhatsAppContImpl(
	guestRepo repositories.GuestRepo,
	guestMessageRepo repositories.GuestMessageRepo,
	settingRepo repositories.SettingRepo,
	userRepo repositories.UsersRepo,
	productRepo repositories.ProductRepo,
	orderRepo repositories.OrderRepo,
	orderPaymentRepo repositories.OrderPaymentRepo,
	customerRepo repositories.CustomerRepo,
	tenantUsageRepo repositories.TenantUsageRepo,
	n8nServ services.N8NServ,
	whatsAppConnectionRepo repositories.WhatsAppConnectionRepo,
	db *gorm.DB,
) *WhatsAppContImpl {
	return &WhatsAppContImpl{
		GuestRepo:              guestRepo,
		GuestMessageRepo:       guestMessageRepo,
		SettingRepo:            settingRepo,
		UserRepo:               userRepo,
		ProductRepo:            productRepo,
		OrderRepo:              orderRepo,
		OrderPaymentRepo:       orderPaymentRepo,
		CustomerRepo:           customerRepo,
		TenantUsageRepo:        tenantUsageRepo,
		N8NServ:                n8nServ,
		WhatsAppConnectionRepo: whatsAppConnectionRepo,
		Db:                     db,
	}
}

// WhatsApp Cloud API webhook payload structures
type WhatsAppWebhookPayload struct {
	Object string            `json:"object"`
	Entry  []WhatsAppEntry   `json:"entry"`
}

type WhatsAppEntry struct {
	ID      string              `json:"id"`
	Changes []WhatsAppChange    `json:"changes"`
}

type WhatsAppChange struct {
	Value WhatsAppChangeValue `json:"value"`
	Field string              `json:"field"`
}

type WhatsAppChangeValue struct {
	MessagingProduct string             `json:"messaging_product"`
	Metadata         WhatsAppMetadata   `json:"metadata"`
	Contacts         []WhatsAppContact  `json:"contacts"`
	Messages         []WhatsAppMessage  `json:"messages"`
	Statuses         []WhatsAppStatus   `json:"statuses"`
}

type WhatsAppMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type WhatsAppContact struct {
	Profile WhatsAppProfile `json:"profile"`
	WaID    string          `json:"wa_id"`
}

type WhatsAppProfile struct {
	Name string `json:"name"`
}

type WhatsAppMessage struct {
	From      string          `json:"from"`
	ID        string          `json:"id"`
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Text      *WhatsAppText   `json:"text,omitempty"`
}

type WhatsAppText struct {
	Body string `json:"body"`
}

type WhatsAppStatus struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Timestamp    string `json:"timestamp"`
	RecipientID  string `json:"recipient_id"`
	Errors       []WhatsAppStatusError `json:"errors"`
	Conversation *WhatsAppConversation `json:"conversation,omitempty"`
	Pricing      *WhatsAppPricing      `json:"pricing,omitempty"`
}

type WhatsAppStatusError struct {
	Code    int    `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message"`
	ErrorData *struct {
		Details string `json:"details"`
	} `json:"error_data,omitempty"`
}

type WhatsAppConversation struct {
	ID                  string `json:"id"`
	ExpirationTimestamp string `json:"expiration_timestamp"`
	Origin              *struct {
		Type string `json:"type"`
	} `json:"origin,omitempty"`
}

type WhatsAppPricing struct {
	Billable     bool   `json:"billable"`
	PricingModel string `json:"pricing_model"`
	Category     string `json:"category"`
}

// VerifyWebhook handles Meta's webhook verification (GET request)
// Meta sends: ?hub.mode=subscribe&hub.verify_token=...&hub.challenge=...
func (cont *WhatsAppContImpl) VerifyWebhook(ctx *gin.Context) {
	schema := ctx.Param("schema")
	mode := ctx.Query("hub.mode")
	token := ctx.Query("hub.verify_token")
	challenge := ctx.Query("hub.challenge")

	if mode != "subscribe" {
		ctx.JSON(403, gin.H{"error": "invalid mode"})
		return
	}

	// Try per-tenant verify token first
	verifyToken := ""
	if schema != "" {
		settings, err := cont.SettingRepo.GetByGroupAndSubGroupName(cont.Db, schema, "integration", "WhatsApp")
		if err == nil {
			for _, s := range settings {
				if s.Name == "whatsapp-verify-token" {
					verifyToken = s.Value
					break
				}
			}
		}
	}

	// Fallback to global env
	if verifyToken == "" {
		verifyToken = os.Getenv("WHATSAPP_VERIFY_TOKEN")
	}

	if verifyToken == "" || token != verifyToken {
		log.Printf("[WhatsApp Verify] invalid token for schema=%s", schema)
		ctx.JSON(403, gin.H{"error": "invalid verify token"})
		return
	}

	log.Printf("[WhatsApp Verify] ✅ webhook verified for schema=%s", schema)
	ctx.String(200, challenge)
}

// Webhook handles incoming WhatsApp messages (POST request from Meta)
func (cont *WhatsAppContImpl) Webhook(ctx *gin.Context) {
	schema := ctx.Param("schema")
	if schema == "" {
		log.Printf("[WhatsApp Webhook] schema required")
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	var payload WhatsAppWebhookPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		log.Printf("[WhatsApp Webhook] bind error: %v", err)
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	// Only handle whatsapp_business_account events
	if payload.Object != "whatsapp_business_account" {
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}
			if len(change.Value.Statuses) > 0 {
				cont.handleStatuses(schema, change.Value.Statuses)
			}
			for _, msg := range change.Value.Messages {
				if msg.Type != "text" || msg.Text == nil {
					continue
				}
				cont.handleIncomingMessage(schema, msg.From, msg.Text.Body, change.Value.Contacts)
			}
		}
	}

	ctx.JSON(200, gin.H{"status": "ok"})
}

func (cont *WhatsAppContImpl) handleStatuses(schema string, statuses []WhatsAppStatus) {
	user, err := cont.UserRepo.FindByUsernameOrEmail(cont.Db, schema, "Tenant")
	if err != nil || user == nil {
		log.Printf("[WhatsApp Status] tenant not found for schema=%s: %v", schema, err)
		return
	}

	var tenantID uuid.UUID
	if user.Tenant != nil {
		tenantID = user.Tenant.TenantID
	}

	for _, st := range statuses {
		errorParts := make([]string, 0, len(st.Errors))
		for _, e := range st.Errors {
			part := strings.TrimSpace(e.Message)
			if part == "" {
				part = strings.TrimSpace(e.Title)
			}
			if e.ErrorData != nil && strings.TrimSpace(e.ErrorData.Details) != "" {
				if part != "" {
					part += " - "
				}
				part += strings.TrimSpace(e.ErrorData.Details)
			}
			if part != "" {
				errorParts = append(errorParts, part)
			}
		}
		errorText := strings.Join(errorParts, " | ")

		log.Printf(
			"[WhatsApp Status] schema=%s recipient=%s status=%s wamid=%s conversation_id=%s pricing_category=%s errors=%s",
			schema,
			st.RecipientID,
			st.Status,
			st.ID,
			func() string {
				if st.Conversation == nil {
					return ""
				}
				return st.Conversation.ID
			}(),
			func() string {
				if st.Pricing == nil {
					return ""
				}
				return st.Pricing.Category
			}(),
			errorText,
		)

		if strings.TrimSpace(st.RecipientID) == "" || tenantID == uuid.Nil {
			continue
		}

		guest, err := cont.GuestRepo.FindByPlatformChatID(cont.Db, schema, st.RecipientID)
		if err != nil || guest == nil {
			continue
		}

		eventData := map[string]interface{}{
			"event": "message_status",
			"data": map[string]interface{}{
				"guest_id":      guest.ID.String(),
				"guest_name":    guest.Name,
				"platform":      "whatsapp",
				"message_id":    st.ID,
				"recipient_id":  st.RecipientID,
				"status":        st.Status,
				"timestamp":     st.Timestamp,
				"error_message": errorText,
			},
		}
		eventJSON, _ := json.Marshal(eventData)
		h := helpers.GetChatHub()
		payload := string(eventJSON)
		h.BroadcastToGuest(user.UserID.String(), guest.ID.String(), payload)
		h.BroadcastToTenant(user.UserID.String(), payload)
	}
}

// handleIncomingMessage processes a single incoming WhatsApp text message
func (cont *WhatsAppContImpl) handleIncomingMessage(schema, from, text string, contacts []WhatsAppContact) {
	// Get tenant info
	user, err := cont.UserRepo.FindByUsernameOrEmail(cont.Db, schema, "Tenant")
	if err != nil || user == nil {
		log.Printf("[WhatsApp] tenant not found for schema=%s: %v", schema, err)
		return
	}

	if user.Tenant == nil || user.Tenant.TenantID == uuid.Nil {
		log.Printf("[WhatsApp] tenant data not found for schema=%s", schema)
		return
	}

	tenantID := user.Tenant.TenantID

	// Get WhatsApp credentials from tenant settings
	waClient := cont.getWhatsAppClient(schema)
	if waClient == nil {
		log.Printf("[WhatsApp] credentials not configured for schema=%s", schema)
		return
	}

	// Extract sender name from contacts
	senderName := from
	for _, c := range contacts {
		if c.WaID == from && c.Profile.Name != "" {
			senderName = c.Profile.Name
			break
		}
	}

	// chatID for WhatsApp = sender's phone number (the `from` field)
	chatID := from

	// Find or create guest
	guest, err := cont.GuestRepo.FindByPlatformChatID(cont.Db, schema, chatID)
	if err != nil {
		// New guest — phone is already known from WhatsApp
		// Normalize phone: WhatsApp sends without '+', e.g. "628123456789"
		phoneFormatted := "+" + from

		guest = &domains.Guest{
			TenantID:         &tenantID,
			Identity:         chatID,
			Username:         from,
			Phone:            phoneFormatted,
			Name:             senderName,
			Platform:         "whatsapp",
			PlatformChatID:   chatID,
			Sosmed: domains.JSONB{
				"wa_id": from,
				"name":  senderName,
			},
			IsActive:          true,
			IsRead:            false,
			IsTakeOver:        false,
			ConversationState: domains.JSONB{"state": "registered"},
		}

		if err := cont.GuestRepo.Create(cont.Db, schema, *guest); err != nil {
			log.Printf("[WhatsApp] failed to create guest: %v", err)
			return
		}

		guest, _ = cont.GuestRepo.FindByPlatformChatID(cont.Db, schema, chatID)

		// Welcome new guest and show menu
		welcome := fmt.Sprintf("👋 Welcome, %s!\n\n", senderName)
		welcome += "What would you like to do?\n\n"
		welcome += "Type 1 - See Products\n"
		welcome += "Type 2 - Create Order\n"
		welcome += "Type 3 - Check Order Status\n"
		welcome += "Type 4 - FAQ\n\n"
		welcome += "Just type 1, 2, 3, or 4"

		cont.sendWABotMessage(waClient, user.UserID, guest.ID, guest.Name, chatID, schema, welcome)
		return
	}

	// Get conversation state
	state := ""
	if guest.ConversationState != nil {
		if s, ok := guest.ConversationState["state"].(string); ok {
			state = s
		}
	}

	// Save and broadcast incoming message
	if text != "" {
		incomingMsg := domains.GuestMessage{
			GuestID:  guest.ID,
			Role:     "user",
			Type:     "text",
			Message:  text,
			Platform: "whatsapp",
			IsHuman:  true,
			IsActive: true,
		}
		if err := cont.GuestMessageRepo.Create(cont.Db, schema, incomingMsg); err != nil {
			log.Printf("[WhatsApp] error saving user message: %v", err)
		}
		cont.wabroadcastMessage(user.UserID, guest.ID, guest.Name, text, "user", false)

		now := time.Now()
		guest.LastMessageAt = &now
		guest.IsRead = false
		cont.GuestRepo.Update(cont.Db, schema, *guest)
	}

	log.Printf("[WhatsApp] schema=%s from=%s state=%s text=%s", schema, from, state, text)

	// Route based on state
	switch state {
	case "creating_order":
		cont.waContinueCreateOrder(waClient, chatID, schema, guest, text, user.UserID)

	case "", "registered", "waiting_for_menu", "browsing_products", "checking_order", "asking_faq":
		switch text {
		case "1":
			cont.waShowProducts(waClient, chatID, schema, guest, user.UserID)
			cont.setGuestState(schema, guest, "browsing_products")

		case "2":
			if !cont.waIsOperationalHoursOpen(schema) {
				cont.sendWABotMessage(waClient, user.UserID, guest.ID, guest.Name, chatID, schema,
					"⏰ Sorry, we are currently outside our operational hours. Please try again during business hours.")
			} else {
				cont.waStartCreateOrder(waClient, chatID, schema, guest)
				cont.setGuestState(schema, guest, "creating_order")
			}

		case "3":
			cont.waShowOrderStatus(waClient, chatID, guest.Phone, schema, -1, user.UserID)
			cont.setGuestState(schema, guest, "checking_order")

		case "4":
			cont.setGuestState(schema, guest, "asking_faq")
			cont.waHandleAIMessage(waClient, chatID, guest, "Hi, I'd like to know the FAQ for this store.", schema, user.UserID, tenantID)

		default:
			if strings.EqualFold(text, "menu") {
				cont.waShowMenu(waClient, chatID, schema, guest, user.UserID)
			} else if ok, oid := parseCheckOrderIntent(text); ok {
				cont.waShowOrderStatus(waClient, chatID, guest.Phone, schema, oid, user.UserID)
			} else if isCreateOrderIntent(text) {
				if !cont.waIsOperationalHoursOpen(schema) {
					cont.sendWABotMessage(waClient, user.UserID, guest.ID, guest.Name, chatID, schema,
						"⏰ Sorry, we are currently outside our operational hours. Please try again during business hours.")
				} else {
					cont.setGuestState(schema, guest, "creating_order")
					cont.waStartCreateOrder(waClient, chatID, schema, guest)
				}
			} else {
				cont.waHandleAIMessage(waClient, chatID, guest, text, schema, user.UserID, tenantID)
			}
		}
	}
}

// getWhatsAppClient builds a WhatsApp client from tenant integration settings
func (cont *WhatsAppContImpl) getWhatsAppClient(schema string) *helpers.WhatsAppClient {
	phoneNumberID := ""
	accessToken := ""

	settings, err := cont.SettingRepo.GetByGroupAndSubGroupName(cont.Db, schema, "integration", "WhatsApp")
	if err == nil {
		for _, s := range settings {
			switch s.Name {
			case "whatsapp-phone-number-id":
				phoneNumberID = s.Value
			case "whatsapp-access-token":
				accessToken = s.Value
			}
		}
	}

	// Fallback to env
	if phoneNumberID == "" {
		phoneNumberID = os.Getenv("WHATSAPP_PHONE_NUMBER_ID")
	}
	if accessToken == "" {
		accessToken = os.Getenv("WHATSAPP_ACCESS_TOKEN")
	}

	if phoneNumberID == "" || accessToken == "" {
		return nil
	}

	return helpers.NewWhatsAppClient(phoneNumberID, accessToken)
}

// setGuestState updates the guest conversation state
func (cont *WhatsAppContImpl) setGuestState(schema string, guest *domains.Guest, state string) {
	if guest.ConversationState == nil {
		guest.ConversationState = domains.JSONB{}
	}
	guest.ConversationState["state"] = state
	cont.GuestRepo.Update(cont.Db, schema, *guest)
}

// waShowMenu sends the main menu
func (cont *WhatsAppContImpl) waShowMenu(waClient *helpers.WhatsAppClient, chatID, schema string, guest *domains.Guest, clientID uuid.UUID) {
	menu := "What would you like to do?\n\n"
	menu += "Type 1 - See Products\n"
	menu += "Type 2 - Create Order\n"
	menu += "Type 3 - Check Order Status\n"
	menu += "Type 4 - FAQ\n\n"
	menu += "Just type 1, 2, 3, or 4"
	cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema, menu)
	cont.setGuestState(schema, guest, "waiting_for_menu")
}

// waShowProducts sends the product list
func (cont *WhatsAppContImpl) waShowProducts(waClient *helpers.WhatsAppClient, chatID, schema string, guest *domains.Guest, clientID uuid.UUID) {
	products, total, err := cont.ProductRepo.GetAll(cont.Db, schema, domains.Pagination{Page: 1, Limit: 10})
	if err != nil || total == 0 {
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"📦 No products available at the moment.\n\nType 'menu' to go back.")
		return
	}

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "https://data.ai-dia.com"
	}

	message := "📦 Our Products:\n\n"
	for i, p := range products {
		message += fmt.Sprintf("%d. %s\n", i+1, p.Name)
		message += fmt.Sprintf("   Price: $%s\n", formatSGDPrice(p.Price))

		if len(p.Images) > 0 && p.Images[0].Image != "" {
			message += fmt.Sprintf("   Image: %s%s\n", appURL, p.Images[0].Image)
		}

		if p.Description != nil && *p.Description != "" {
			message += fmt.Sprintf("   %s\n\n", *p.Description)
		} else {
			message += "\n"
		}
	}
	message += "Type 'menu' to go back to main menu."
	cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema, message)
}

// waShowOrderStatus shows order status
func (cont *WhatsAppContImpl) waShowOrderStatus(waClient *helpers.WhatsAppClient, chatID, phone, schema string, orderID int, clientID uuid.UUID) {
	guest, err := cont.GuestRepo.FindByPlatformChatID(cont.Db, schema, chatID)
	if err != nil || guest == nil {
		waClient.SendMessage(chatID, "📦 No orders found.\n\nType 'menu' to go back.")
		return
	}

	guestPhone := guest.Phone
	phoneNumber := guestPhone
	if len(guestPhone) > 3 && guestPhone[0] == '+' {
		phoneNumber = guestPhone[3:]
	}

	if orderID > 0 {
		order, err := cont.OrderRepo.GetByID(cont.Db, schema, orderID)
		if err != nil || order == nil {
			cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
				fmt.Sprintf("📦 Order #%d not found.\n\nType 'menu' to go back.", orderID))
			return
		}
		cont.waSendOrderStatusMessage(waClient, chatID, schema, guest, []domains.Order{*order}, clientID)
		return
	}

	customer, err := cont.OrderRepo.GetCustomerByPhone(cont.Db, schema, phoneNumber)
	if err != nil {
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"📦 No orders found.\n\nType 'menu' to go back.")
		return
	}

	orders, err := cont.OrderRepo.GetByCustomerID(cont.Db, schema, customer.ID)
	if err != nil || len(orders) == 0 {
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"📦 You have no orders yet.\n\nType 'menu' to go back.")
		return
	}

	if orderID == -1 && len(orders) > 0 {
		orders = orders[:1]
	}

	cont.waSendOrderStatusMessage(waClient, chatID, schema, guest, orders, clientID)
}

// waSendOrderStatusMessage formats and sends order status messages
func (cont *WhatsAppContImpl) waSendOrderStatusMessage(waClient *helpers.WhatsAppClient, chatID, schema string, guest *domains.Guest, orders []domains.Order, clientID uuid.UUID) {
	var storeName string
	cont.Db.Raw(`SELECT bp.business_name FROM public.business_profile bp
		JOIN public.tenant t ON t.tenant_id = bp.tenant_id
		JOIN public.users u ON u.user_id = t.user_id
		WHERE u.tenant_schema = ? LIMIT 1`, schema).Scan(&storeName)
	if storeName == "" {
		storeName = schema
	}

	allProducts, _, _ := cont.ProductRepo.GetAll(cont.Db, schema, domains.Pagination{Page: 1, Limit: 100})
	productNames := make(map[string]string, len(allProducts))
	for _, p := range allProducts {
		productNames[p.ID.String()] = p.Name
	}

	appURL := os.Getenv("APP_URL")

	for _, o := range orders {
		isPaid := o.Payment != nil && string(o.Payment.PaymentStatus) == "Paid"
		if !isPaid {
			continue
		}

		itemParts := ""
		for _, p := range o.Products {
			name := productNames[p.ProductID]
			if name == "" {
				name = "Product"
			}
			if itemParts != "" {
				itemParts += ", "
			}
			itemParts += fmt.Sprintf("%s(%d)", name, p.Quantity)
		}

		service := o.DeliverySubGroupName
		if service == "" || service == "Default" {
			service = "Delivery"
		}
		orderDate := o.CreatedAt.Format("02 Jan 2006")
		detailURL := fmt.Sprintf("%s/orders/%s/%d", appURL, schema, o.ID)

		customerName := ""
		customerPhone := guest.Phone
		if o.Customer != nil {
			customerName = o.Customer.Name
			if o.Customer.PhoneCountryCode != nil && o.Customer.PhoneNumber != nil {
				customerPhone = *o.Customer.PhoneCountryCode + *o.Customer.PhoneNumber
			}
		}

		msg := fmt.Sprintf("Store: %s\n\n", storeName)
		msg += fmt.Sprintf("Order #%d\n\n", o.ID)
		msg += fmt.Sprintf("Order Items: %s\n\n", itemParts)
		msg += fmt.Sprintf("Service: %s (%s)\n\n", service, orderDate)
		msg += fmt.Sprintf("Total amount: $%s\n\n", formatPriceSGD(o.TotalPrice))
		if customerName != "" {
			msg += fmt.Sprintf("Customer: %s %s\n\n", customerName, customerPhone)
		}
		msg += fmt.Sprintf("See order details - %s", detailURL)
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema, msg)
	}

	var pendingMsg string
	for _, o := range orders {
		isPaid := o.Payment != nil && string(o.Payment.PaymentStatus) == "Paid"
		if isPaid {
			continue
		}

		statusEmoji := "⏳"
		switch o.Status {
		case domains.OrderStatusCompleted:
			statusEmoji = "✅"
		case domains.OrderStatusCancelled:
			statusEmoji = "❌"
		case domains.OrderStatusConfirmed:
			statusEmoji = "✔️"
		}

		paymentInfo := "Unpaid"
		if o.Payment != nil {
			switch o.Payment.PaymentStatus {
			case domains.PaymentStatusConfirmingPayment:
				paymentInfo = "Confirming payment"
			case domains.PaymentStatusRefunded:
				paymentInfo = "Refunded"
			case domains.PaymentStatusVoided:
				paymentInfo = "Expired"
			}
		}

		itemParts := ""
		for _, p := range o.Products {
			name := productNames[p.ProductID]
			if name == "" {
				name = "Product"
			}
			if itemParts != "" {
				itemParts += ", "
			}
			itemParts += fmt.Sprintf("%s(%d)", name, p.Quantity)
		}

		detailURL := fmt.Sprintf("%s/orders/%s/%d", appURL, schema, o.ID)
		pendingMsg += fmt.Sprintf("%s Order #%d - %s\n", statusEmoji, o.ID, o.Status)
		if itemParts != "" {
			pendingMsg += fmt.Sprintf("   Items: %s\n", itemParts)
		}
		pendingMsg += fmt.Sprintf("   Total: $%s | %s\n", formatPriceSGD(o.TotalPrice), paymentInfo)
		pendingMsg += fmt.Sprintf("   Details: %s\n\n", detailURL)
	}

	if pendingMsg != "" {
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"📦 Your Orders:\n\n"+pendingMsg+"Type 'menu' to go back.")
	}
}

// waHandleAIMessage forwards message to n8n for AI processing
// tenantID is used for token usage tracking.
func (cont *WhatsAppContImpl) waHandleAIMessage(waClient *helpers.WhatsAppClient, chatID string, guest *domains.Guest, message, schema string, clientID, tenantID uuid.UUID) {
	log.Printf("[WhatsApp/AI] handling message for guest %s: %s", guest.ID, message)

	// Check free token limit when no active subscription
	if !hasActiveSubs(cont.Db, cont.TenantUsageRepo, tenantID) {
		if freeTokensRemaining(cont.Db, cont.TenantUsageRepo, tenantID) <= 0 {
			cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
				"⚠️ This store has reached its free AI message limit.\n\nTo continue using the AI assistant, the store owner needs to upgrade to a paid subscription.\n\nFor assistance, please contact the store directly.")
			return
		}
	}

	guestID := guest.ID
	guestName := guest.Name

	go func() {
		history, _ := cont.GuestMessageRepo.GetLatestMessages(cont.Db, schema, guestID, 10)

		n8nResp, err := cont.N8NServ.ProcessMessage(schema, guestID.String(), chatID, message, "", history)
		if err != nil {
			log.Printf("[WhatsApp/AI] n8n error: %v", err)
			waClient.SendMessage(chatID, "⚠️ Maaf, saya sedang mengalami kendala. Silakan coba lagi nanti.")
			return
		}

		// Deduct tokens from free usage if no active subscription
		if !hasActiveSubs(cont.Db, cont.TenantUsageRepo, tenantID) && n8nResp.UsageTokens > 0 {
			deductFreeTokens(cont.Db, cont.TenantUsageRepo, tenantID, n8nResp.UsageTokens)
			log.Printf("[Token] deducted %d tokens for tenant %s", n8nResp.UsageTokens, tenantID)
		}

		if strings.Contains(n8nResp.Reply, "__ACTION:SHOW_PRODUCTS__") {
			freshGuest, err := cont.GuestRepo.FindByPlatformChatID(cont.Db, schema, chatID)
			if err == nil && freshGuest != nil {
				cont.waShowProducts(waClient, chatID, schema, freshGuest, clientID)
				cont.setGuestState(schema, freshGuest, "browsing_products")
			}
			return
		}

		if strings.Contains(n8nResp.Reply, "__ACTION:CREATE_ORDER__") {
			freshGuest, err := cont.GuestRepo.FindByPlatformChatID(cont.Db, schema, chatID)
			if err == nil && freshGuest != nil {
				cont.waStartCreateOrder(waClient, chatID, schema, freshGuest)
			}
			return
		}

		if strings.Contains(n8nResp.Reply, "__ACTION:CHECK_ORDER") {
			orderID := 0
			if strings.Contains(n8nResp.Reply, "__ACTION:CHECK_ORDER:RECENT__") {
				orderID = -1
			} else if m := regexp.MustCompile(`__ACTION:CHECK_ORDER:(\d+)__`).FindStringSubmatch(n8nResp.Reply); len(m) > 1 {
				orderID, _ = strconv.Atoi(m[1])
			}
			freshGuest, err := cont.GuestRepo.FindByPlatformChatID(cont.Db, schema, chatID)
			if err == nil && freshGuest != nil {
				cont.waShowOrderStatus(waClient, chatID, freshGuest.Phone, schema, orderID, clientID)
			}
			return
		}

		aiMsg := domains.GuestMessage{
			GuestID:  guestID,
			Role:     "assistant",
			Type:     "text",
			Message:  n8nResp.Reply,
			Platform: "whatsapp",
			IsHuman:  false,
			IsActive: true,
		}
		cont.GuestMessageRepo.Create(cont.Db, schema, aiMsg)
		cont.wabroadcastMessage(clientID, guestID, guestName, n8nResp.Reply, "assistant", false)

		if err := waClient.SendMessage(chatID, n8nResp.Reply); err != nil {
			log.Printf("[WhatsApp/AI] error sending reply: %v", err)
		} else {
			log.Printf("[WhatsApp/AI] ✅ reply sent to %s", chatID)
		}
	}()
}

// sendWABotMessage sends a message, saves it to DB, and broadcasts to SSE
func (cont *WhatsAppContImpl) sendWABotMessage(waClient *helpers.WhatsAppClient, clientID, guestID uuid.UUID, guestName, chatID, schema, message string) {
	waClient.SendMessage(chatID, message)

	msg := domains.GuestMessage{
		GuestID:  guestID,
		Role:     "assistant",
		Type:     "text",
		Message:  message,
		Platform: "whatsapp",
		IsHuman:  false,
		IsActive: true,
	}
	cont.GuestMessageRepo.Create(cont.Db, schema, msg)
	cont.wabroadcastMessage(clientID, guestID, guestName, message, "assistant", false)
}

// wabroadcastMessage pushes a new message event to SSE clients
func (cont *WhatsAppContImpl) wabroadcastMessage(clientID, guestID uuid.UUID, guestName, message, role string, isHuman bool) {
	sgt, _ := time.LoadLocation("Asia/Singapore")
	eventData := map[string]interface{}{
		"event": "new_message",
		"data": map[string]interface{}{
			"guest_id":   guestID.String(),
			"guest_name": guestName,
			"message":    message,
			"role":       role,
			"is_human":   isHuman,
			"created_at": time.Now().In(sgt).Format(time.RFC3339),
		},
	}
	eventJSON, _ := json.Marshal(eventData)
	payload := string(eventJSON)

	h := helpers.GetChatHub()
	h.BroadcastToGuest(clientID.String(), guestID.String(), payload)
	h.BroadcastToTenant(clientID.String(), payload)
}

// waIsOperationalHoursOpen checks if the store is currently open
func (cont *WhatsAppContImpl) waIsOperationalHoursOpen(schema string) bool {
	settings, err := cont.SettingRepo.GetByGroupAndSubGroupName(cont.Db, schema, "ai_prompt", "AI Operational")
	if err != nil {
		return true
	}
	hoursJSON := ""
	for _, s := range settings {
		if s.Name == "ai-operational-prompt" {
			hoursJSON = s.Value
			break
		}
	}
	if hoursJSON == "" {
		return true
	}
	open, err := helpers.IsWithinOperationalHours(hoursJSON, "Asia/Singapore")
	if err != nil {
		return true
	}
	return open
}

// GetAIContextForSchema returns AI context for n8n (same data as Telegram)
func (cont *WhatsAppContImpl) GetAIContextForSchema(ctx *gin.Context) {
	schema := ctx.Param("schema")
	if schema == "" {
		ctx.JSON(400, gin.H{"error": "schema required"})
		return
	}

	getPrompt := func(subGroup, name string) string {
		settings, err := cont.SettingRepo.GetByGroupAndSubGroupName(cont.Db, schema, "ai_prompt", subGroup)
		if err != nil {
			return ""
		}
		for _, s := range settings {
			if s.Name == name {
				return s.Value
			}
		}
		return ""
	}

	prompts := map[string]string{
		"product":     getPrompt("AI Product", "ai-product-prompt"),
		"delivery":    getPrompt("AI Delivery", "ai-delivery-prompt"),
		"operational": getPrompt("AI Operational", "ai-operational-prompt"),
		"about_store": getPrompt("AI About Store", "ai-about-store-prompt"),
		"faq":         getPrompt("AI FAQ", "ai-faq-prompt"),
	}

	type ProductItem struct {
		Name        string  `json:"name"`
		Price       float64 `json:"price"`
		Description string  `json:"description"`
		OutOfStock  bool    `json:"out_of_stock"`
	}
	var products []ProductItem
	dbProducts, total, err := cont.ProductRepo.GetAll(cont.Db, schema, domains.Pagination{Page: 1, Limit: 100})
	if err == nil && total > 0 {
		for _, p := range dbProducts {
			desc := ""
			if p.Description != nil {
				desc = *p.Description
			}
			products = append(products, ProductItem{
				Name:        p.Name,
				Price:       p.Price,
				Description: desc,
				OutOfStock:  p.IsOutOfStock,
			})
		}
	}

	type DeliveryZoneItem struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	var deliveryZones []DeliveryZoneItem
	rawSettings, err := cont.SettingRepo.GetByGroupName(cont.Db, schema, "delivery")
	if err == nil {
		for _, s := range rawSettings {
			if s.Name == "sub-group-name" {
				deliveryZones = append(deliveryZones, DeliveryZoneItem{Name: s.Value})
			}
		}
	}

	ctx.JSON(200, gin.H{
		"prompts":        prompts,
		"products":       products,
		"delivery_zones": deliveryZones,
	})
}

// VerifyWebhookGlobal godoc
// @Summary      Verify WhatsApp Webhook (Global)
// @Description  Meta calls this endpoint once when registering the webhook URL in the Meta App Dashboard. Responds with hub.challenge to confirm ownership. No authentication required.
// @Tags         WhatsApp Webhook
// @Produce      plain
// @Param        hub.mode          query  string  true  "Must be 'subscribe'"
// @Param        hub.verify_token  query  string  true  "Must match META_VERIFY_TOKEN in server config"
// @Param        hub.challenge     query  string  true  "Challenge string to echo back"
// @Success      200
// @Failure      403  {object}  helpers.ApiResponse
// @Router       /webhook/whatsapp [get]
func (cont *WhatsAppContImpl) VerifyWebhookGlobal(ctx *gin.Context) {
	mode := ctx.Query("hub.mode")
	token := ctx.Query("hub.verify_token")
	challenge := ctx.Query("hub.challenge")

	if mode != "subscribe" {
		ctx.JSON(403, gin.H{"error": "invalid mode"})
		return
	}

	verifyToken := os.Getenv("META_VERIFY_TOKEN")
	if verifyToken == "" {
		// Fallback ke token lama agar tidak breaking
		verifyToken = os.Getenv("WHATSAPP_VERIFY_TOKEN")
	}

	if verifyToken == "" || token != verifyToken {
		log.Printf("[WhatsApp Global Verify] token tidak cocok")
		ctx.JSON(403, gin.H{"error": "invalid verify token"})
		return
	}

	log.Printf("[WhatsApp Global Verify] ✅ webhook verified")
	ctx.String(200, challenge)
}

// WebhookGlobal godoc
// @Summary      Receive WhatsApp Webhook (Global)
// @Description  Receives all incoming WhatsApp messages from Meta for all tenants. Routes each message to the correct tenant based on phone_number_id in the payload. No authentication required — called by Meta only.
// @Tags         WhatsApp Webhook
// @Accept       json
// @Produce      json
// @Success      200  {object}  helpers.ApiResponse
// @Router       /webhook/whatsapp [post]
func (cont *WhatsAppContImpl) WebhookGlobal(ctx *gin.Context) {
	var payload WhatsAppWebhookPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		log.Printf("[WhatsApp Global] bind error: %v", err)
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	if payload.Object != "whatsapp_business_account" {
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}

			// Routing: cari tenant berdasarkan phone_number_id
			phoneNumberID := change.Value.Metadata.PhoneNumberID
			if phoneNumberID == "" {
				continue
			}

			conn, err := cont.WhatsAppConnectionRepo.FindByPhoneNumberID(cont.Db, phoneNumberID)
			if err != nil || conn == nil {
				log.Printf("[WhatsApp Global] tidak ada tenant untuk phone_number_id=%s", phoneNumberID)
				continue
			}

			if len(change.Value.Statuses) > 0 {
				cont.handleStatuses(conn.TenantSchema, change.Value.Statuses)
			}

			for _, msg := range change.Value.Messages {
				if msg.Type != "text" || msg.Text == nil {
					continue
				}
				cont.handleIncomingMessage(conn.TenantSchema, msg.From, msg.Text.Body, change.Value.Contacts)
			}
		}
	}

	ctx.JSON(200, gin.H{"status": "ok"})
}

var _ = (*WhatsAppContImpl)(nil)
