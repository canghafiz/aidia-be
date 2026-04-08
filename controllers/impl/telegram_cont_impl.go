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

type TelegramContImpl struct {
	GuestRepo        repositories.GuestRepo
	GuestMessageRepo repositories.GuestMessageRepo
	SettingRepo      repositories.SettingRepo
	UserRepo         repositories.UsersRepo
	ProductRepo      repositories.ProductRepo
	OrderRepo        repositories.OrderRepo
	OrderPaymentRepo repositories.OrderPaymentRepo
	CustomerRepo     repositories.CustomerRepo
	N8NServ          services.N8NServ
	Db               *gorm.DB
}

func NewTelegramContImpl(
	guestRepo repositories.GuestRepo,
	guestMessageRepo repositories.GuestMessageRepo,
	settingRepo repositories.SettingRepo,
	userRepo repositories.UsersRepo,
	productRepo repositories.ProductRepo,
	orderRepo repositories.OrderRepo,
	orderPaymentRepo repositories.OrderPaymentRepo,
	customerRepo repositories.CustomerRepo,
	n8nServ services.N8NServ,
	db *gorm.DB,
) *TelegramContImpl {
	return &TelegramContImpl{
		GuestRepo:        guestRepo,
		GuestMessageRepo: guestMessageRepo,
		SettingRepo:      settingRepo,
		UserRepo:         userRepo,
		ProductRepo:      productRepo,
		OrderRepo:        orderRepo,
		OrderPaymentRepo: orderPaymentRepo,
		CustomerRepo:     customerRepo,
		N8NServ:          n8nServ,
		Db:               db,
	}
}

// TelegramWebhookRequest represents incoming Telegram webhook payload
type TelegramWebhookRequest struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		MessageID int      `json:"message_id"`
		From      *User    `json:"from"`
		Chat      *Chat    `json:"chat"`
		Date      int64    `json:"date"`
		Text      string   `json:"text"`
		Contact   *Contact `json:"contact"`
	} `json:"message"`
	CallbackQuery *struct {
		ID      string `json:"id"`
		From    *User  `json:"from"`
		Data    string `json:"data"`
		Message *struct {
			MessageID int   `json:"message_id"`
			Chat      *Chat `json:"chat"`
		} `json:"message"`
	} `json:"callback_query"`
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

// isValidPhoneNumber validates phone number format
func isValidPhoneNumber(phone string) bool {
	matched, _ := regexp.MatchString(`^\+\d{10,15}$`, phone)
	return matched
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
	text := payload.Message.Text

	// Get bot token
	setting, _ := cont.SettingRepo.GetByGroupAndSubGroupName(cont.Db, schema, "integration", "Telegram")
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

	tgClient := helpers.NewTelegramClient(botToken)

	// Get tenant info
	user, err := cont.UserRepo.FindByUsernameOrEmail(cont.Db, schema, "Tenant")
	if err != nil || user == nil {
		log.Printf("[Telegram Webhook] tenant not found: %v", err)
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	if user.Tenant == nil || user.Tenant.TenantID == uuid.Nil {
		log.Printf("[Telegram Webhook] tenant data not found")
		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	tenantID := user.Tenant.TenantID

	// Find or create guest
	guest, err := cont.GuestRepo.FindByPlatformChatID(cont.Db, schema, chatID)
	if err != nil {
		// Create new guest
		fullName := payload.Message.From.FirstName
		if payload.Message.From.LastName != "" {
			fullName += " " + payload.Message.From.LastName
		}

		guest = &domains.Guest{
			TenantID:         &tenantID,
			Identity:         chatID,
			Username:         payload.Message.From.Username,
			Phone:            "",
			Name:             fullName,
			PlatformChatID:   chatID,
			PlatformUsername: payload.Message.From.Username,
			Sosmed: domains.JSONB{
				"id":         float64(payload.Message.From.ID),
				"first_name": payload.Message.From.FirstName,
				"last_name":  payload.Message.From.LastName,
				"username":   payload.Message.From.Username,
				"is_bot":     payload.Message.From.IsBot,
			},
			IsActive:          true,
			IsRead:            false,
			IsTakeOver:        false,
			ConversationState: domains.JSONB{"state": "waiting_for_phone"},
		}

		if err := cont.GuestRepo.Create(cont.Db, schema, *guest); err != nil {
			log.Printf("[Telegram Webhook] failed to create guest: %v", err)
			ctx.JSON(200, gin.H{"status": "ok"})
			return
		}

		guest, _ = cont.GuestRepo.FindByPlatformChatID(cont.Db, schema, chatID)

		// NEW GUEST - Ask for phone number
		tgClient.SendMessage(chatID, "👋 Welcome! To complete your registration, please send your phone number.\n\nFormat: +628123456789\n\n⚠️ This is required to continue.")

		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	// Get conversation state
	state := ""
	if guest.ConversationState != nil {
		if s, ok := guest.ConversationState["state"].(string); ok {
			state = s
		}
	}

	// EXISTING GUEST - Check if phone already exists
	if guest.Phone == "" || guest.Phone == " " {
		// Waiting for phone number
		if text != "" {
			if isValidPhoneNumber(text) {
				// Update guest phone
				guest.Phone = text
				if guest.ConversationState == nil {
					guest.ConversationState = domains.JSONB{}
				}
				guest.ConversationState["state"] = "registered"
				cont.GuestRepo.Update(cont.Db, schema, *guest)
				log.Printf("[Telegram Webhook] ✅ Guest phone updated: %s", guest.Phone)

				// Send success message + MENU
				menu := "✅ Phone number updated successfully!\n\n"
				menu += "Your registration is complete! 🎉\n\n"
				menu += "What would you like to do?\n\n"
				menu += "Type 1 - See Products\n"
				menu += "Type 2 - Create Order\n"
				menu += "Type 3 - Check Order Status\n"
				menu += "Type 4 - FAQ\n\n"
				menu += "Just type 1, 2, 3, or 4"

				tgClient.SendMessage(chatID, menu)
			} else {
				// Invalid phone format
				tgClient.SendMessage(chatID, "❌ Invalid phone number format.\n\nPlease use format: +628123456789\n\nExample: +628123456789")
			}
		}

		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	// PHONE EXISTS - Save and broadcast every incoming message before routing
	if text != "" {
		incomingMsg := domains.GuestMessage{
			GuestID:  guest.ID,
			Role:     "user",
			Type:     "text",
			Message:  text,
			IsHuman:  true, // actual human typing in Telegram
			IsActive: true,
		}
		if err := cont.GuestMessageRepo.Create(cont.Db, schema, incomingMsg); err != nil {
			log.Printf("[Telegram] Error saving user message: %v", err)
		}
		cont.broadcastMessage(user.UserID, guest.ID, guest.Name, text, "user", false)

		now := time.Now()
		guest.LastMessageAt = &now
		guest.IsRead = false
		cont.GuestRepo.Update(cont.Db, schema, *guest)
	}

	// Handle menu navigation
	log.Printf("[Telegram] DEBUG: guest.Phone='%s', state='%s', text='%s'", guest.Phone, state, text)
	
	if state == "" || state == "registered" {
		log.Printf("[Telegram] Guest found, state: %s, text: %s", state, text)
		
		// Check if user is selecting menu
		if text == "2" {
			log.Printf("[Telegram] User selected Create Order")
			if !cont.isOperationalHoursOpen(schema) {
				cont.sendBotMessage(tgClient, user.UserID, guest.ID, guest.Name, chatID, schema, "⏰ Sorry, we are currently outside our operational hours. Please try again during business hours.")
				ctx.JSON(200, gin.H{"status": "ok"})
				return
			}
			// Create Order - Start order creation flow
			cont.startCreateOrder(tgClient, chatID, schema, guest)

			// Update state to creating_order
			if guest.ConversationState == nil {
				guest.ConversationState = domains.JSONB{}
			}
			guest.ConversationState["state"] = "creating_order"
			cont.GuestRepo.Update(cont.Db, schema, *guest)

			log.Printf("[Telegram] Order creation started, state updated")

			ctx.JSON(200, gin.H{"status": "ok"})
			return
		}
		
		// Show menu for other inputs
		log.Printf("[Telegram] Showing menu to user")
		menu := "✅ Registration complete!\n\n"
		menu += "What would you like to do?\n\n"
		menu += "Type 1 - See Products\n"
		menu += "Type 2 - Create Order\n"
		menu += "Type 3 - Check Order Status\n"
		menu += "Type 4 - FAQ\n\n"
		menu += "Just type 1, 2, 3, or 4"

		cont.sendBotMessage(tgClient, user.UserID, guest.ID, guest.Name, chatID, schema, menu)

		// Update state
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		guest.ConversationState["state"] = "waiting_for_menu"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		ctx.JSON(200, gin.H{"status": "ok"})
		return
	}

	// Handle menu selection based on state
	switch state {
	case "creating_order":
		// User is in order creation flow - handle order input FIRST!
		log.Printf("[Telegram] User in creating_order state, input: %s", text)
		cont.continueCreateOrder(tgClient, chatID, schema, guest, text, user.UserID)
		
	case "waiting_for_menu", "browsing_products", "checking_order", "asking_faq":
		// User can choose menu from these states
		if text == "1" {
			// Show products
			cont.showProducts(tgClient, chatID, schema, guest, user.UserID)

			if guest.ConversationState == nil {
				guest.ConversationState = domains.JSONB{}
			}
			guest.ConversationState["state"] = "browsing_products"
			cont.GuestRepo.Update(cont.Db, schema, *guest)
		} else if text == "2" {
			// Create Order - check operational hours first
			if !cont.isOperationalHoursOpen(schema) {
				cont.sendBotMessage(tgClient, user.UserID, guest.ID, guest.Name, chatID, schema, "⏰ Sorry, we are currently outside our operational hours. Please try again during business hours.")
			} else {
				cont.startCreateOrder(tgClient, chatID, schema, guest)

				if guest.ConversationState == nil {
					guest.ConversationState = domains.JSONB{}
				}
				guest.ConversationState["state"] = "creating_order"
				cont.GuestRepo.Update(cont.Db, schema, *guest)
			}
		} else if text == "3" {
			// Check order status — show most recent by default
			cont.showOrderStatus(tgClient, chatID, guest.Phone, schema, -1, user.UserID)

			if guest.ConversationState == nil {
				guest.ConversationState = domains.JSONB{}
			}
			guest.ConversationState["state"] = "checking_order"
			cont.GuestRepo.Update(cont.Db, schema, *guest)
		} else if text == "4" {
			// FAQ - set state asking_faq then immediately let AI respond using FAQ prompt context
			if guest.ConversationState == nil {
				guest.ConversationState = domains.JSONB{}
			}
			guest.ConversationState["state"] = "asking_faq"
			cont.GuestRepo.Update(cont.Db, schema, *guest)

			cont.handleAIMessage(tgClient, chatID, guest, "Hi, I'd like to know the FAQ for this store.", schema, user.UserID)
		} else if strings.EqualFold(text, "menu") {
			// Show main menu
			cont.showMenu(tgClient, chatID, schema, guest, user.UserID)
		} else if isShowProductsIntent(text) {
			cont.showProducts(tgClient, chatID, schema, guest, user.UserID)
			cont.setGuestState(schema, guest, "browsing_products")
		} else if ok, oid := parseCheckOrderIntent(text); ok {
			cont.showOrderStatus(tgClient, chatID, guest.Phone, schema, oid, user.UserID)
		} else if isCreateOrderIntent(text) {
			if !cont.isOperationalHoursOpen(schema) {
				cont.sendBotMessage(tgClient, user.UserID, guest.ID, guest.Name, chatID, schema, "⏰ Sorry, we are currently outside our operational hours. Please try again during business hours.")
			} else {
				if guest.ConversationState == nil {
					guest.ConversationState = domains.JSONB{}
				}
				guest.ConversationState["state"] = "creating_order"
				cont.GuestRepo.Update(cont.Db, schema, *guest)
				cont.startCreateOrder(tgClient, chatID, schema, guest)
			}
		} else {
			// Free-form text → AI responds with combined prompt from all sections
			cont.handleAIMessage(tgClient, chatID, guest, text, schema, user.UserID)
		}

	case "registered":
		// Should not reach here, show menu
		cont.showMenu(tgClient, chatID, schema, guest, user.UserID)
	}

	ctx.JSON(200, gin.H{"status": "ok"})
}

// showMenu shows the main menu
func (cont *TelegramContImpl) showMenu(tgClient *helpers.TelegramClient, chatID, schema string, guest *domains.Guest, clientID uuid.UUID) {
	menu := "✅ What would you like to do?\n\n"
	menu += "Type 1 - See Products\n"
	menu += "Type 2 - Create Order\n"
	menu += "Type 3 - Check Order Status\n"
	menu += "Type 4 - FAQ\n\n"
	menu += "Just type 1, 2, 3, or 4"

	cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, menu)

	if guest.ConversationState == nil {
		guest.ConversationState = domains.JSONB{}
	}
	guest.ConversationState["state"] = "waiting_for_menu"
}

// parseCheckOrderIntent detects order status check intent and extracts optional order ID.
// Returns (isIntent, orderID) where orderID: 0=all, -1=most recent, >0=specific order.
func parseCheckOrderIntent(text string) (bool, int) {
	lower := strings.ToLower(text)

	// Must contain order-related noun
	orderNouns := []string{
		"order", "pesanan", "pemesanan", "belanjaan", "transaksi",
	}
	hasOrderWord := false
	for _, w := range orderNouns {
		if strings.Contains(lower, w) {
			hasOrderWord = true
			break
		}
	}
	if !hasOrderWord {
		return false, 0
	}

	// Must also have a qualifying word showing intent to check/view
	qualifiers := []string{
		"check", "status", "my", "see", "view", "track", "history",
		"cek", "lihat", "gimana", "mana", "sampai", "nyampe",
		"dimana", "progress", "update", "recent", "latest", "last",
		"udah", "sudah", "belum", "selesai", "done", "#",
	}
	hasQualifier := false
	for _, q := range qualifiers {
		if strings.Contains(lower, q) {
			hasQualifier = true
			break
		}
	}
	if !hasQualifier {
		return false, 0
	}

	// Specific order ID: "order #4", "order 4", "#4"
	if m := regexp.MustCompile(`(?:order\s*#?|#\s*)(\d+)`).FindStringSubmatch(lower); len(m) > 1 {
		id, _ := strconv.Atoi(m[1])
		if id > 0 {
			return true, id
		}
	}

	// "all" intent → show all orders
	allKeywords := []string{"all", "semua", "seluruh", "every", "list all", "all order", "semua pesanan"}
	for _, kw := range allKeywords {
		if strings.Contains(lower, kw) {
			return true, 0
		}
	}

	// Default: show most recent order only
	return true, -1
}

// isCheckOrderIntent is kept for backward compatibility
func isCheckOrderIntent(text string) bool {
	ok, _ := parseCheckOrderIntent(text)
	return ok
}

// isCreateOrderIntent detects create order intent from free-form text.
// Strategy: must have order-action verb AND order-noun (or just strong single phrases).
func isCreateOrderIntent(text string) bool {
	lower := strings.ToLower(text)

	// Strong single phrases — langsung true
	strongPhrases := []string{
		"place order", "make order", "create order", "mau order", "mau pesan",
		"mau beli", "mau beli", "buat pesanan", "pesan sekarang", "order sekarang",
		"i want to order", "i wanna order", "i'd like to order",
		"i want to buy", "i wanna buy", "pengen order", "pengen pesan", "pengen beli",
		"ingin order", "ingin pesan", "ingin beli", "want to purchase",
		"order dong", "pesan dong", "beli dong", "order ya", "pesan ya",
		"bisa order", "bisa pesan", "bisa beli",
	}
	for _, kw := range strongPhrases {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	// Word-group: action verb + order noun
	actionVerbs := []string{
		"order", "pesan", "beli", "purchase", "buy", "checkout",
	}
	orderNouns := []string{
		"makanan", "minuman", "produk", "item", "barang", "food", "drink",
	}
	for _, verb := range actionVerbs {
		for _, noun := range orderNouns {
			if strings.Contains(lower, verb) && strings.Contains(lower, noun) {
				return true
			}
		}
	}

	return false
}

// isShowProductsIntent detects intent to view products/menu from free-form text.
// Strategy: must have view-verb AND product-noun, or strong single phrases.
func isShowProductsIntent(text string) bool {
	lower := strings.ToLower(text)

	// Strong single phrases — langsung true
	strongPhrases := []string{
		"see product", "show product", "view product", "lihat produk",
		"see menu", "show menu", "view menu", "lihat menu",
		"what do you have", "what do you sell", "what do you offer",
		"what's on the menu", "what is on the menu",
		"apa saja produk", "apa aja produk", "apa produk",
		"ada menu apa", "ada produk apa", "ada apa aja", "ada apa saja",
		"daftar produk", "daftar menu", "tampilkan produk", "tampilkan menu",
		"kasih lihat produk", "kasih lihat menu",
		"your product", "your menu",
	}
	for _, kw := range strongPhrases {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	// Word-group: view verb + product noun
	viewVerbs := []string{
		"see", "show", "view", "display", "browse", "check out",
		"lihat", "liat", "tampilkan", "kasih", "cek",
	}
	productNouns := []string{
		"product", "produk", "menu", "makanan", "minuman", "item",
		"barang", "food", "drink", "catalogue", "catalog",
	}
	for _, verb := range viewVerbs {
		for _, noun := range productNouns {
			if strings.Contains(lower, verb) && strings.Contains(lower, noun) {
				return true
			}
		}
	}

	return false
}

// showProducts shows product list from database
func (cont *TelegramContImpl) showProducts(tgClient *helpers.TelegramClient, chatID, schema string, guest *domains.Guest, clientID uuid.UUID) {
	// Get products from database
	products, total, err := cont.ProductRepo.GetAll(cont.Db, schema, domains.Pagination{Page: 1, Limit: 10})
	if err != nil || total == 0 {
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "📦 No products available at the moment.\n\nPlease check back later!\n\nType 1, 2, or 3 to choose from menu.")
		return
	}

	// Get APP_URL from environment
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "https://data.ai-dia.com" // Default fallback
	}

	message := "📦 **Our Products:**\n\n"
	for i, p := range products {
		message += fmt.Sprintf("%d. %s\n", i+1, p.Name)
		message += fmt.Sprintf("   Price: $%s\n", formatSGDPrice(p.Price))

		// Add product image if exists
		if len(p.Images) > 0 && p.Images[0].Image != "" {
			imageURL := fmt.Sprintf("%s%s", appURL, p.Images[0].Image)
			message += fmt.Sprintf("   [️ View Image](%s)\n", imageURL)
		}

		if p.Description != nil && *p.Description != "" {
			message += fmt.Sprintf("   %s\n\n", *p.Description)
		} else {
			message += "\n"
		}
	}

	message += "\nType 1, 2, or 3 to choose from menu, or type 'menu' to go back to main menu."

	cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, message)
}

// showOrderStatus shows order status from database.
// orderID: 0 = all orders, -1 = most recent only, >0 = specific order ID.
func (cont *TelegramContImpl) showOrderStatus(tgClient *helpers.TelegramClient, chatID, phone, schema string, orderID int, clientID uuid.UUID) {
	guest, err := cont.GuestRepo.FindByPlatformChatID(cont.Db, schema, chatID)
	if err != nil || guest == nil {
		tgClient.SendMessage(chatID, "📦 No orders found.\n\nType 'menu' to go back.")
		return
	}

	guestPhone := guest.Phone
	phoneNumber := guestPhone
	if len(guestPhone) > 3 && guestPhone[0] == '+' {
		phoneNumber = guestPhone[3:]
	}

	// Specific order by ID — no phone lookup needed
	if orderID > 0 {
		order, err := cont.OrderRepo.GetByID(cont.Db, schema, orderID)
		if err != nil || order == nil {
			cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, fmt.Sprintf("📦 Order #%d not found.\n\nType 'menu' to go back.", orderID))
			return
		}
		cont.sendOrderStatusMessage(tgClient, chatID, schema, guest, []domains.Order{*order}, clientID)
		return
	}

	customer, err := cont.OrderRepo.GetCustomerByPhone(cont.Db, schema, phoneNumber)
	if err != nil {
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "📦 No orders found.\n\nType 'menu' to go back.")
		return
	}

	orders, err := cont.OrderRepo.GetByCustomerID(cont.Db, schema, customer.ID)
	if err != nil || len(orders) == 0 {
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "📦 You have no orders yet.\n\nType 'menu' to go back.")
		return
	}

	// Most recent only
	if orderID == -1 && len(orders) > 0 {
		orders = orders[:1]
	}

	cont.sendOrderStatusMessage(tgClient, chatID, schema, guest, orders, clientID)
}

// sendOrderStatusMessage formats and sends order status messages.
// Paid orders → one bubble each. Non-paid → grouped in one message.
func (cont *TelegramContImpl) sendOrderStatusMessage(tgClient *helpers.TelegramClient, chatID, schema string, guest *domains.Guest, orders []domains.Order, clientID uuid.UUID) {
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
	hasPaid := false

	for _, o := range orders {
		isPaid := o.Payment != nil && string(o.Payment.PaymentStatus) == "Paid"
		if !isPaid {
			continue
		}
		hasPaid = true

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
			customerPhone = o.Customer.PhoneCountryCode + o.Customer.PhoneNumber
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
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, msg)
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
		pendingMsg += fmt.Sprintf("%s *Order #%d* - %s\n", statusEmoji, o.ID, o.Status)
		if itemParts != "" {
			pendingMsg += fmt.Sprintf("   Items: %s\n", itemParts)
		}
		pendingMsg += fmt.Sprintf("   Total: $%s | %s\n", formatPriceSGD(o.TotalPrice), paymentInfo)
		pendingMsg += fmt.Sprintf("   Details: %s\n\n", detailURL)
	}

	if pendingMsg != "" {
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "📦 *Your Orders:*\n\n"+pendingMsg+"Type 'menu' to go back.")
	} else if !hasPaid {
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "📦 No orders found.\n\nType 'menu' to go back.")
	}
}

// getAIPromptSetting reads one AI prompt setting from the tenant schema
func (cont *TelegramContImpl) getAIPromptSetting(schema, subGroupName, settingName string) string {
	settings, err := cont.SettingRepo.GetByGroupAndSubGroupName(cont.Db, schema, "ai_prompt", subGroupName)
	if err != nil {
		return ""
	}
	for _, s := range settings {
		if s.Name == settingName {
			return s.Value
		}
	}
	return ""
}

// handleAIMessage forwards user message to n8n. N8N will call /internal/{schema}/ai-context
// to fetch prompts + live data, then build the full prompt and call OpenAI.
// clientID = user_id of the tenant owner (used as hub key for SSE broadcasts).
func (cont *TelegramContImpl) handleAIMessage(tgClient *helpers.TelegramClient, chatID string, guest *domains.Guest, message, schema string, clientID uuid.UUID) {
	log.Printf("[AI] Handling message for guest %s: %s", guest.ID, message)

	guestID := guest.ID
	guestName := guest.Name

	go func() {
		history, _ := cont.GuestMessageRepo.GetLatestMessages(cont.Db, schema, guestID, 10)

		// Send to n8n — prompt is empty, n8n fetches context from /internal/{schema}/ai-context
		n8nResp, err := cont.N8NServ.ProcessMessage(schema, guestID.String(), chatID, message, "", history)
		if err != nil {
			log.Printf("[AI] n8n error: %v", err)
			tgClient.SendMessage(chatID, "⚠️ Maaf, saya sedang mengalami kendala. Silakan coba lagi nanti.")
			return
		}

		// Check if AI signals create order intent
		if strings.Contains(n8nResp.Reply, "__ACTION:CREATE_ORDER__") {
			log.Printf("[AI] Create order intent detected for chat %s", chatID)
			freshGuest, err := cont.GuestRepo.FindByPlatformChatID(cont.Db, schema, chatID)
			if err == nil && freshGuest != nil {
				cont.startCreateOrder(tgClient, chatID, schema, freshGuest)
			}
			return
		}

		// Check if AI signals check order status intent
		// Supports: __ACTION:CHECK_ORDER__ / __ACTION:CHECK_ORDER:RECENT__ / __ACTION:CHECK_ORDER:4__
		if strings.Contains(n8nResp.Reply, "__ACTION:CHECK_ORDER") {
			log.Printf("[AI] Check order intent detected for chat %s", chatID)
			orderID := 0
			if strings.Contains(n8nResp.Reply, "__ACTION:CHECK_ORDER:RECENT__") {
				orderID = -1
			} else if m := regexp.MustCompile(`__ACTION:CHECK_ORDER:(\d+)__`).FindStringSubmatch(n8nResp.Reply); len(m) > 1 {
				orderID, _ = strconv.Atoi(m[1])
			}
			freshGuest, err := cont.GuestRepo.FindByPlatformChatID(cont.Db, schema, chatID)
			if err == nil && freshGuest != nil {
				cont.showOrderStatus(tgClient, chatID, freshGuest.Phone, schema, orderID, clientID)
			}
			return
		}

		aiMsg := domains.GuestMessage{
			GuestID:  guestID,
			Role:     "assistant",
			Type:     "text",
			Message:  n8nResp.Reply,
			IsHuman:  false,
			IsActive: true,
		}
		cont.GuestMessageRepo.Create(cont.Db, schema, aiMsg)

		// Broadcast AI reply to SSE clients
		cont.broadcastMessage(clientID, guestID, guestName, n8nResp.Reply, "assistant", false)

		if _, err := tgClient.SendMessage(chatID, n8nResp.Reply); err != nil {
			log.Printf("[AI] Error sending reply: %v", err)
		} else {
			log.Printf("[AI] ✅ Reply sent to %s (%s)", chatID, guestName)
		}
	}()
}

// sendBotMessage sends a text reply to Telegram, saves it to guest_message, and broadcasts to SSE.
// Use this instead of bare tgClient.SendMessage for all bot replies that should appear in the dashboard.
func (cont *TelegramContImpl) sendBotMessage(tgClient *helpers.TelegramClient, clientID, guestID uuid.UUID, guestName, chatID, schema, message string) {
	tgClient.SendMessage(chatID, message)

	msg := domains.GuestMessage{
		GuestID:  guestID,
		Role:     "assistant",
		Type:     "text",
		Message:  message,
		IsHuman:  false,
		IsActive: true,
	}
	cont.GuestMessageRepo.Create(cont.Db, schema, msg)
	cont.broadcastMessage(clientID, guestID, guestName, message, "assistant", false)
}

// isOperationalHoursOpen returns true if the current Singapore time is within the configured operational hours.
func (cont *TelegramContImpl) isOperationalHoursOpen(schema string) bool {
	hoursJSON := cont.getAIPromptSetting(schema, "AI Operational", "ai-operational-prompt")
	if hoursJSON == "" {
		return true // no setting = always open
	}
	open, err := helpers.IsWithinOperationalHours(hoursJSON, "Asia/Singapore")
	if err != nil {
		return true // parse error = treat as open
	}
	return open
}

// broadcastMessage pushes a new message event to SSE clients (detail view + conversations list).
func (cont *TelegramContImpl) broadcastMessage(clientID, guestID uuid.UUID, guestName, message, role string, isHuman bool) {
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

// GetAIContextForSchema godoc
// @Summary      Get AI Context for Schema (Internal API for n8n)
// @Description  Returns AI prompts (5 sections) + live products + delivery zones for a tenant. Called by n8n to build the full AI prompt.
// @Tags         Internal
// @Produce      json
// @Param        schema  path  string  true  "Tenant Schema"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Router       /api/v1/internal/telegram/{schema}/ai-context [get]
func (cont *TelegramContImpl) GetAIContextForSchema(ctx *gin.Context) {
	schema := ctx.Param("schema")
	if schema == "" {
		ctx.JSON(400, gin.H{"error": "schema required"})
		return
	}

	// --- AI Prompts (5 sections from tenant setting) ---
	prompts := map[string]string{
		"product":     cont.getAIPromptSetting(schema, "AI Product", "ai-product-prompt"),
		"delivery":    cont.getAIPromptSetting(schema, "AI Delivery", "ai-delivery-prompt"),
		"operational": cont.getAIPromptSetting(schema, "AI Operational", "ai-operational-prompt"),
		"about_store": cont.getAIPromptSetting(schema, "AI About Store", "ai-about-store-prompt"),
		"faq":         cont.getAIPromptSetting(schema, "AI FAQ", "ai-faq-prompt"),
	}

	// --- Products ---
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

	// --- Delivery Zones ---
	type DeliveryZoneItem struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	var deliveryZones []DeliveryZoneItem
	rawSettings, err := cont.SettingRepo.GetByGroupName(cont.Db, schema, "delivery")
	if err == nil {
		zones := domains.ToDeliverySetting(rawSettings)
		for _, z := range zones {
			if z.IsVisible {
				deliveryZones = append(deliveryZones, DeliveryZoneItem{
					Name:        z.Name,
					Description: z.Description,
				})
			}
		}
	}

	ctx.JSON(200, gin.H{
		"schema":        schema,
		"prompts":       prompts,
		"products":      products,
		"delivery_zones": deliveryZones,
	})
}

// GetPublicOrderDetail is a public endpoint (no auth) that shows order details.
// Redirects to Stripe hosted invoice if available; otherwise renders a full HTML order page.
func (cont *TelegramContImpl) GetPublicOrderDetail(ctx *gin.Context) {
	schema := ctx.Param("schema")
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || schema == "" {
		ctx.Data(400, "text/html; charset=utf-8", []byte("<h3>Invalid order link.</h3>"))
		return
	}

	order, err := cont.OrderRepo.GetByID(cont.Db, schema, id)
	if err != nil || order == nil {
		ctx.Data(404, "text/html; charset=utf-8", []byte("<h3>Order not found.</h3>"))
		return
	}

	// Build product name map
	allProducts, _, _ := cont.ProductRepo.GetAll(cont.Db, schema, domains.Pagination{Page: 1, Limit: 100})
	productNames := make(map[string]string, len(allProducts))
	for _, p := range allProducts {
		productNames[p.ID.String()] = p.Name
	}

	// Build items rows
	itemRows := ""
	for _, p := range order.Products {
		name := productNames[p.ProductID]
		if name == "" {
			name = "Product"
		}
		itemRows += fmt.Sprintf(
			`<tr><td>%s</td><td style="text-align:center">%d</td><td style="text-align:right">$%.2f</td></tr>`,
			name, p.Quantity, p.TotalPrice)
	}

	// Payment status badge color
	paymentStatus := "Unpaid"
	paymentColor := "#e53935"
	paymentInvoiceBtn := ""
	if order.Payment != nil {
		paymentStatus = string(order.Payment.PaymentStatus)
		switch order.Payment.PaymentStatus {
		case domains.PaymentStatusPaid:
			paymentColor = "#2e7d32"
		case domains.PaymentStatusConfirmingPayment:
			paymentColor = "#f57c00"
		case domains.PaymentStatusVoided:
			paymentColor = "#757575"
		}
		if order.Payment.StripeSessionURL != nil && *order.Payment.StripeSessionURL != "" {
			paymentInvoiceBtn = fmt.Sprintf(
				`<a href="%s" style="display:inline-block;margin-top:16px;padding:12px 24px;background:#635bff;color:#fff;text-decoration:none;border-radius:8px;font-weight:600">Pay Invoice</a>`,
				*order.Payment.StripeSessionURL)
		}
	}

	// Order status badge color
	orderColor := "#f57c00"
	switch order.Status {
	case domains.OrderStatusConfirmed:
		orderColor = "#1565c0"
	case domains.OrderStatusCompleted:
		orderColor = "#2e7d32"
	case domains.OrderStatusCancelled:
		orderColor = "#b71c1c"
	}

	// Customer info
	customerInfo := ""
	if order.Customer != nil {
		customerInfo = fmt.Sprintf(`<p style="margin:4px 0"><strong>Customer:</strong> %s</p>
<p style="margin:4px 0"><strong>Phone:</strong> %s%s</p>`,
			order.Customer.Name, order.Customer.PhoneCountryCode, order.Customer.PhoneNumber)
	}

	// Store name
	var storeName string
	cont.Db.Raw(`SELECT bp.business_name FROM public.business_profile bp
		JOIN public.tenant t ON t.tenant_id = bp.tenant_id
		JOIN public.users u ON u.user_id = t.user_id
		WHERE u.tenant_schema = ? LIMIT 1`, schema).Scan(&storeName)
	if storeName == "" {
		storeName = schema
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Order #%d — %s</title>
  <style>
    *{box-sizing:border-box;margin:0;padding:0}
    body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#f5f5f5;color:#333;padding:24px 16px}
    .card{background:#fff;border-radius:12px;max-width:480px;margin:0 auto;overflow:hidden;box-shadow:0 2px 12px rgba(0,0,0,.08)}
    .header{padding:20px 24px;border-bottom:1px solid #f0f0f0}
    .store{font-size:13px;color:#888;margin-bottom:4px}
    .title{font-size:20px;font-weight:700}
    .section{padding:16px 24px;border-bottom:1px solid #f0f0f0}
    .label{font-size:12px;color:#888;text-transform:uppercase;letter-spacing:.5px;margin-bottom:8px}
    .badge{display:inline-block;padding:3px 10px;border-radius:20px;font-size:12px;font-weight:600;color:#fff}
    table{width:100%%;border-collapse:collapse;font-size:14px}
    th{text-align:left;font-size:12px;color:#888;padding:0 0 8px;border-bottom:1px solid #f0f0f0}
    th:not(:first-child){text-align:center}th:last-child{text-align:right}
    td{padding:8px 0;vertical-align:top}
    .total-row td{font-weight:700;font-size:15px;padding-top:12px;border-top:1px solid #f0f0f0}
    .footer{padding:16px 24px;text-align:center;font-size:12px;color:#aaa}
  </style>
</head>
<body>
<div class="card">
  <div class="header">
    <div class="store">%s</div>
    <div class="title">Order #%d</div>
    <div style="margin-top:8px;font-size:13px;color:#888">%s</div>
  </div>

  <div class="section">
    <div class="label">Status</div>
    <span class="badge" style="background:%s">%s</span>
    &nbsp;
    <span class="badge" style="background:%s">%s</span>
  </div>

  <div class="section">
    <div class="label">Order Items</div>
    <table>
      <thead><tr><th>Item</th><th>Qty</th><th>Amount</th></tr></thead>
      <tbody>%s</tbody>
      <tfoot><tr class="total-row"><td colspan="2">Total</td><td style="text-align:right">$%.2f</td></tr></tfoot>
    </table>
  </div>

  <div class="section">
    <div class="label">Delivery</div>
    <p style="font-size:14px">%s</p>
    <p style="font-size:13px;color:#888;margin-top:4px">%s, %s</p>
  </div>

  <div class="section">
    <div class="label">Customer</div>
    <div style="font-size:14px">%s</div>
  </div>

  <div style="padding:20px 24px;text-align:center">
    %s
  </div>

  <div class="footer">Order placed on %s</div>
</div>
</body>
</html>`,
		order.ID, storeName,
		storeName,
		order.ID,
		order.CreatedAt.Format("02 Jan 2006, 15:04"),
		orderColor, string(order.Status),
		paymentColor, paymentStatus,
		itemRows,
		order.TotalPrice,
		order.DeliverySubGroupName,
		order.StreetAddress, order.PostalCode,
		customerInfo,
		paymentInvoiceBtn,
		order.CreatedAt.Format("02 Jan 2006, 15:04"),
	)

	ctx.Data(200, "text/html; charset=utf-8", []byte(html))
}

// formatPrice formats price to IDR format
func formatPrice(price float64) string {
	return fmt.Sprintf("%.0f", price)
}

// formatSGDPrice formats price to SGD format
func formatSGDPrice(price float64) string {
	return fmt.Sprintf("%.2f", price)
}

var _ interface{} = (*TelegramContImpl)(nil)
