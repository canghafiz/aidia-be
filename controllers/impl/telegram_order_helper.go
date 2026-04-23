package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/invoiceitem"
	stripecustomer "github.com/stripe/stripe-go/v81/customer"
	stripeinvoice "github.com/stripe/stripe-go/v81/invoice"
)

// ParsedProduct is the structured product selection extracted by AI.
type ParsedProduct struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// startCreateOrder starts the order creation flow
func (cont *TelegramContImpl) startCreateOrder(tgClient *helpers.TelegramClient, chatID, schema string, guest *domains.Guest, clientID uuid.UUID) {
	log.Printf("[Order] Starting order creation for guest %s", guest.ID)

	// Get products
	products, total, _ := cont.ProductRepo.GetAll(cont.Db, schema, domains.Pagination{Page: 1, Limit: 20})
	if total == 0 {
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "📦 No products available at the moment.\n\nType 'menu' to go back to main menu.")
		return
	}

	// Show products
	message := "🛒 *Silakan pilih produk yang ingin dipesan:*\n\n"
	for i, p := range products {
		message += fmt.Sprintf("*%d. %s* - $%s\n", i+1, p.Name, formatPriceSGD(p.Price))
		if p.Description != nil && *p.Description != "" {
			message += fmt.Sprintf("   _%s_\n", *p.Description)
		}
	}
	message += "\nJust tell me what you'd like, e.g.:\n"
	message += "• _'1 Mie Tek Tek'_\n"
	message += "• _'2 Fried Rice and 1 Iced Tea'_\n"
	message += "\nType 'menu' to cancel"

	cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, message)

	// Update conversation state
	if guest.ConversationState == nil {
		guest.ConversationState = domains.JSONB{}
	}
	guest.ConversationState["state"] = "creating_order"
	guest.ConversationState["order_step"] = "products"
	guest.ConversationState["guest_phone"] = guest.Phone

	log.Printf("[Order] Updated state: creating_order, order_step: products")
	cont.GuestRepo.Update(cont.Db, schema, *guest)
}

// continueCreateOrder continues the order creation flow based on current step
func (cont *TelegramContImpl) continueCreateOrder(tgClient *helpers.TelegramClient, chatID, schema string, guest *domains.Guest, text string, clientID uuid.UUID) {
	// Check if user wants to cancel
	if strings.EqualFold(text, "menu") {
		log.Printf("[Order] User cancelled order creation")
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		guest.ConversationState["state"] = "waiting_for_menu"
		delete(guest.ConversationState, "order_step")
		delete(guest.ConversationState, "products_parsed")
		delete(guest.ConversationState, "customer_name")
		delete(guest.ConversationState, "customer_email")
		delete(guest.ConversationState, "guest_phone")
		delete(guest.ConversationState, "address")
		delete(guest.ConversationState, "postal_code")
		cont.GuestRepo.Update(cont.Db, schema, *guest)
		cont.showMenu(tgClient, chatID, schema, guest, clientID)
		return
	}

	orderStep := ""
	if guest.ConversationState != nil {
		if s, ok := guest.ConversationState["order_step"].(string); ok {
			orderStep = s
		}
	}

	log.Printf("[Order] Continue order creation, step: %s, input: %s", orderStep, text)

	if orderStep == "" {
		log.Printf("[Order] ERROR: order_step is empty! ConversationState: %v", guest.ConversationState)
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "⚠️ Order session expired. Please type 'menu' to start over.")
		return
	}

	switch orderStep {
	case "products":
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}

		// Fetch product list so AI knows what's available
		allProducts, _, _ := cont.ProductRepo.GetAll(cont.Db, schema, domains.Pagination{Page: 1, Limit: 100})

		// Let AI understand the user's intent — any language, any phrasing
		parsed, err := parseProductsWithAI(text, allProducts)
		if err != nil {
			log.Printf("[Order] AI parse failed (%v), falling back to text match", err)
			parsed = parseProductsFallback(text, allProducts)
		}

		if len(parsed) == 0 {
			msg := "⚠️ I couldn't understand your product selection.\n\n"
			msg += "Please mention the product name and quantity, e.g.:\n"
			msg += "• _'1 Mie Tek Tek'_\n"
			msg += "• _'2 Fried Rice and 1 Iced Tea'_\n\n"
			msg += "Type 'menu' to cancel"
			cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, msg)
			return
		}

		// Build confirmation message
		confirmMsg := "✅ *Produk yang akan dipesan:*\n"
		for _, p := range parsed {
			for _, prod := range allProducts {
				if prod.ID.String() == p.ProductID {
					confirmMsg += fmt.Sprintf("• %dx %s - $%s\n", p.Quantity, prod.Name, formatPriceSGD(prod.Price*float64(p.Quantity)))
					break
				}
			}
		}

		// Save as JSON — already structured, no parsing needed at finalize
		parsedJSON, _ := json.Marshal(parsed)
		guest.ConversationState["products_parsed"] = string(parsedJSON)
		guest.ConversationState["order_step"] = "name"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, confirmMsg+"\n*What is your full name?*\n\nType 'menu' to cancel")

	case "name":
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		guest.ConversationState["customer_name"] = text
		guest.ConversationState["order_step"] = "email"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "✅ Name saved!\n\n*Email address?* (for invoice delivery)\n\nExample: test@example.com\n\nType 'menu' to cancel")

	case "email":
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		if !isValidEmail(text) {
			cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "❌ Invalid email format. Please enter a valid email address.\n\nExample: john@example.com\n\nType 'menu' to cancel")
			return
		}
		guest.ConversationState["customer_email"] = text
		guest.ConversationState["order_step"] = "address"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "✅ Email saved!\n\n*Delivery address?* (Street, building, etc.)\n\nType 'menu' to cancel")

	case "address":
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		guest.ConversationState["address"] = text
		guest.ConversationState["order_step"] = "postal_code"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "✅ Address saved!\n\n*Postal code?*\n\nType 'menu' to cancel")

	case "postal_code":
		cont.finalizeCreateOrder(tgClient, chatID, schema, guest, text, clientID)

	default:
		log.Printf("[Order] Unknown order_step: %s", orderStep)
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "⚠️ Invalid step. Please type 'menu' to start over.")
	}
}

// finalizeCreateOrder creates the order and saves to database
func (cont *TelegramContImpl) finalizeCreateOrder(tgClient *helpers.TelegramClient, chatID, schema string, guest *domains.Guest, postalCode string, clientID uuid.UUID) {
	if guest.ConversationState == nil {
		guest.ConversationState = domains.JSONB{}
	}

	productsParsedJSON, _ := guest.ConversationState["products_parsed"].(string)
	customerName, _ := guest.ConversationState["customer_name"].(string)
	customerEmail, _ := guest.ConversationState["customer_email"].(string)
	address, _ := guest.ConversationState["address"].(string)
	guestPhone, _ := guest.ConversationState["guest_phone"].(string)

	log.Printf("[Order] Finalizing order: name=%s, email=%s, phone=%s, address=%s, postal=%s",
		customerName, customerEmail, guestPhone, address, postalCode)

	// Extract country code and phone number from guest phone
	phoneCountryCode := "+62"
	phoneNumber := guestPhone
	if len(guestPhone) > 3 && guestPhone[0] == '+' {
		phoneCountryCode = guestPhone[:3]
		phoneNumber = guestPhone[3:]
	}

	// Start transaction
	tx := cont.Db.Begin()
	if tx.Error != nil {
		log.Printf("[Order] ERROR: Failed to start transaction: %v", tx.Error)
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "❌ Error creating order. Please try again.")
		return
	}

	// 1. Find or create customer
	customer, err := cont.CustomerRepo.GetByPhone(tx, schema, phoneCountryCode, phoneNumber)
	if err != nil {
		customer = &domains.Customer{
			Name:             customerName,
			PhoneCountryCode: &phoneCountryCode,
			PhoneNumber:      &phoneNumber,
			AccountType:      "Telegram",
		}
		customer, err = cont.CustomerRepo.Create(tx, schema, *customer)
		if err != nil {
			tx.Rollback()
			log.Printf("[Order] ERROR: Failed to create customer: %v", err)
			cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "❌ Error creating customer. Please try again.")
			return
		}
		log.Printf("[Order] Created new customer: ID=%d, Name=%s", customer.ID, customer.Name)
	} else {
		log.Printf("[Order] Found existing customer: ID=%d, Name=%s", customer.ID, customer.Name)
	}

	// 2. Build order products from AI-parsed result
	allProducts, _, _ := cont.ProductRepo.GetAll(tx, schema, domains.Pagination{Page: 1, Limit: 100})
	var orderProducts []domains.OrderProduct
	var totalPrice float64
	var productsSummary string

	if productsParsedJSON != "" {
		var parsed []ParsedProduct
		if err := json.Unmarshal([]byte(productsParsedJSON), &parsed); err == nil {
			for _, p := range parsed {
				for _, prod := range allProducts {
					if prod.ID.String() == p.ProductID && p.Quantity > 0 {
						itemTotal := prod.Price * float64(p.Quantity)
						totalPrice += itemTotal
						orderProducts = append(orderProducts, domains.OrderProduct{
							ProductID:  prod.ID.String(),
							Quantity:   p.Quantity,
							TotalPrice: itemTotal,
						})
						productsSummary += fmt.Sprintf("%dx %s, ", p.Quantity, prod.Name)
						break
					}
				}
			}
		}
	}
	productsSummary = strings.TrimSuffix(productsSummary, ", ")

	if len(orderProducts) == 0 {
		tx.Rollback()
		log.Printf("[Order] ERROR: No valid products in order")
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "❌ No valid products in order. Please try again.")
		return
	}

	// 3. Create order
	order := &domains.Order{
		CustomerID:           customer.ID,
		TotalPrice:           totalPrice,
		Status:               domains.OrderStatusPending,
		DeliverySubGroupName: "Default",
		StreetAddress:        address,
		PostalCode:           postalCode,
	}

	order, err = cont.OrderRepo.Create(tx, schema, *order)
	if err != nil {
		tx.Rollback()
		log.Printf("[Order] ERROR: Failed to create order: %v", err)
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "❌ Error creating order. Please try again.")
		return
	}
	log.Printf("[Order] Created order: ID=%d, CustomerID=%d, Total=%f", order.ID, customer.ID, totalPrice)

	// 4. Create order products
	for i := range orderProducts {
		orderProducts[i].OrderID = order.ID
	}
	err = cont.OrderRepo.CreateOrderProducts(tx, schema, orderProducts)
	if err != nil {
		tx.Rollback()
		log.Printf("[Order] ERROR: Failed to create order products: %v", err)
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "❌ Error creating order products. Please try again.")
		return
	}
	log.Printf("[Order] Created %d order products", len(orderProducts))

	// 5. Create Stripe invoice
	log.Printf("[Order] Creating Stripe invoice for order ID=%d", order.ID)
	stripeInvoiceID, stripeInvoiceURL, err := cont.createStripeCheckoutSession(schema, order, customer, customerEmail)
	if err != nil {
		log.Printf("[Order] ERROR: Failed to create Stripe invoice: %v", err)
		stripeInvoiceID = nil
		stripeInvoiceURL = nil
	}

	if stripeInvoiceID != nil {
		log.Printf("[Order] Stripe invoice created: ID=%s, URL=%v", *stripeInvoiceID, stripeInvoiceURL)
	}
	if stripeInvoiceURL == nil {
		log.Printf("[Order] WARNING: Stripe invoice URL is nil!")
	}

	// 6. Create or update order payment
	stripePaymentStatus := ""
	if stripeInvoiceID != nil {
		stripePaymentStatus = "open"
	}

	existingPayment, _ := cont.OrderPaymentRepo.GetByOrderID(tx, schema, order.ID)
	if existingPayment != nil && existingPayment.ID != uuid.Nil {
		tx.Table(schema+".order_payments").
			Where("id = ?", existingPayment.ID).
			Updates(map[string]interface{}{
				"payment_session_id":     stripeInvoiceID,
				"payment_session_url":    stripeInvoiceURL,
				"payment_gateway_status": stripePaymentStatus,
				"payment_invoice_id":     stripeInvoiceID,
			})
		log.Printf("[Order] Updated existing order payment: ID=%s", existingPayment.ID)
	} else {
		orderPayment := &domains.OrderPayment{
			OrderID:              order.ID,
			PaymentStatus:        domains.PaymentStatusUnpaid,
			PaymentMethod:        "stripe",
			PaymentGateway:       "stripe",
			TotalPrice:           totalPrice,
			ExpireAt:             order.CreatedAt.Add(15 * time.Minute),
			PaymentSessionID:     stripeInvoiceID,
			PaymentSessionURL:    stripeInvoiceURL,
			PaymentGatewayStatus: &stripePaymentStatus,
			PaymentInvoiceID:     stripeInvoiceID,
		}
		_, err = cont.OrderPaymentRepo.Create(tx, schema, *orderPayment)
		if err != nil {
			tx.Rollback()
			log.Printf("[Order] ERROR: Failed to create order payment: %v", err)
			cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "❌ Error creating payment. Please try again.")
			return
		}
		log.Printf("[Order] Created new order payment")
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		log.Printf("[Order] ERROR: Failed to commit transaction: %v", err)
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "❌ Error finalizing order. Please try again.")
		return
	}

	log.Printf("[Order] ✅ Order created successfully: ID=%d", order.ID)

	// 7. Send success message
	paymentLink := "Invoice will be sent separately"
	if stripeInvoiceURL != nil && *stripeInvoiceURL != "" {
		paymentLink = *stripeInvoiceURL
	} else if stripeInvoiceID != nil {
		paymentLink = fmt.Sprintf("https://invoice.stripe.com/i/%s", *stripeInvoiceID)
	}

	summary := "🎉 *Order Created!*\n\n"
	summary += "✅ *Order details:*\n"
	summary += fmt.Sprintf("- Order ID: #%d\n", order.ID)
	summary += fmt.Sprintf("- Items: %s\n", productsSummary)
	summary += fmt.Sprintf("- Name: %s\n", customerName)
	summary += fmt.Sprintf("- Phone: %s%s\n", phoneCountryCode, phoneNumber)
	summary += fmt.Sprintf("- Address: %s\n", address)
	summary += fmt.Sprintf("- Postal code: %s\n", postalCode)
	summary += fmt.Sprintf("- Total: $%s\n", formatPriceSGD(totalPrice))
	summary += "\n💳 *Pay Now:*\n"
	summary += fmt.Sprintf("%s\n\n", paymentLink)
	summary += "⏰ *Order expires in 15 minutes!*\n\n"
	summary += "Type 'menu' to go back to the main menu."

	cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, summary)

	// 8. Reset state
	guest.ConversationState["state"] = "waiting_for_menu"
	delete(guest.ConversationState, "order_step")
	delete(guest.ConversationState, "products_parsed")
	delete(guest.ConversationState, "customer_name")
	delete(guest.ConversationState, "customer_email")
	delete(guest.ConversationState, "guest_phone")
	delete(guest.ConversationState, "address")
	delete(guest.ConversationState, "postal_code")
	cont.GuestRepo.Update(cont.Db, schema, *guest)
}

// parseProductsWithAI calls OpenAI to extract product selections from any natural language text.
// The AI understands any language and phrasing — "1 mie tek tek", "pengen nasi goreng 2 porsi", etc.
func parseProductsWithAI(userText string, products []domains.Product) ([]ParsedProduct, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not configured")
	}

	// Build product list for AI reference
	productList := ""
	for i, p := range products {
		productList += fmt.Sprintf("%d. %s (id: %s)\n", i+1, p.Name, p.ID.String())
	}

	systemPrompt := `You are a product order parser. Your job is to extract which products a customer wants to order and in what quantity, based on their message.

Available products are listed as "number. name (id: uuid)".

Rules:
- Match product names flexibly: partial names, typos, synonyms all count.
- Default quantity is 1 if not specified.
- Respond ONLY with a valid JSON array. No explanation, no extra text.
- Each item must be: {"product_id":"<exact-uuid>","quantity":<positive-integer>}
- If no products can be identified, return: []

Examples:
User: "i want 1 mie tek tek" → [{"product_id":"<uuid>","quantity":1}]
User: "pengen nasi goreng 2 porsi sama es teh" → [{"product_id":"<uuid-nasi>","quantity":2},{"product_id":"<uuid-es-teh>","quantity":1}]
User: "kasih 3 yang pertama" → [{"product_id":"<uuid-first-product>","quantity":3}]`

	userPrompt := "Available products:\n" + productList + "\nCustomer message: " + userText

	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"max_tokens":  300,
		"temperature": 0,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode OpenAI response: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("empty OpenAI response")
	}

	content := strings.TrimSpace(result.Choices[0].Message.Content)
	log.Printf("[Order/AI] Product parse response: %s", content)

	// Extract JSON array from response (guard against AI adding prose)
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start < 0 || end <= start {
		return []ParsedProduct{}, nil
	}

	var parsed []ParsedProduct
	if err := json.Unmarshal([]byte(content[start:end+1]), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse AI JSON: %w", err)
	}
	return parsed, nil
}

// detectIntentWithAI uses OpenAI to understand the meaning of the user's message
// and classify it into one of: "SHOW_PRODUCTS", "CREATE_ORDER", "CHECK_ORDER", "OTHER".
// Works in any language and any phrasing — OpenAI understands the semantic intent.
// For CHECK_ORDER, also returns orderID: -1=most recent, 0=all, >0=specific order ID.
func detectIntentWithAI(text string) (string, int) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "OTHER", 0
	}

	systemPrompt := `You are an intent classifier for a restaurant ordering chatbot.
Understand the MEANING of the user's message in ANY language or phrasing, then classify it into exactly one intent:

SHOW_PRODUCTS - user wants to see, browse, view, or ask about the menu, products, food, or drinks available
CREATE_ORDER - user wants to place, make, or create a new order; or directly states what they want to buy/order
CHECK_ORDER - user wants to check, view, track, or ask about the status of their existing order(s)
OTHER - anything else (greetings, complaints, store questions, etc.)

For CHECK_ORDER responses:
- Specific order number mentioned → CHECK_ORDER:NUMBER  (e.g. CHECK_ORDER:5)
- Asking about all orders → CHECK_ORDER:ALL
- Default (recent/latest) → CHECK_ORDER:RECENT

Respond with ONLY the intent string. No explanation, no extra text.`

	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": text},
		},
		"max_tokens":  15,
		"temperature": 0,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "OTHER", 0
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "OTHER", 0
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Choices) == 0 {
		return "OTHER", 0
	}

	content := strings.TrimSpace(result.Choices[0].Message.Content)
	log.Printf("[Intent/AI] '%s' → %s", text, content)

	if strings.HasPrefix(content, "CHECK_ORDER") {
		parts := strings.SplitN(content, ":", 2)
		if len(parts) == 2 {
			suffix := strings.TrimSpace(parts[1])
			if suffix == "ALL" {
				return "CHECK_ORDER", 0
			}
			if id, err := strconv.Atoi(suffix); err == nil && id > 0 {
				return "CHECK_ORDER", id
			}
		}
		return "CHECK_ORDER", -1
	}

	switch content {
	case "SHOW_PRODUCTS":
		return "SHOW_PRODUCTS", 0
	case "CREATE_ORDER":
		return "CREATE_ORDER", 0
	default:
		return "OTHER", 0
	}
}

// parseProductsFallback is a last-resort parser used when OpenAI is unavailable.
// It matches product names in the text and extracts preceding digits as quantity.
func parseProductsFallback(input string, products []domains.Product) []ParsedProduct {
	var result []ParsedProduct
	inputLower := strings.ToLower(input)

	// Try "number:quantity" structured format first
	entries := strings.Split(input, ",")
	for _, entry := range entries {
		parts := strings.Split(strings.TrimSpace(entry), ":")
		if len(parts) == 2 {
			num, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			qty, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil && num >= 1 && num <= len(products) && qty >= 1 {
				result = append(result, ParsedProduct{
					ProductID: products[num-1].ID.String(),
					Quantity:  qty,
				})
			}
		}
	}
	if len(result) > 0 {
		return result
	}

	// Name-based matching
	for _, product := range products {
		nameLower := strings.ToLower(product.Name)
		if strings.Contains(inputLower, nameLower) {
			qty := extractQtyBeforeName(inputLower, nameLower)
			result = append(result, ParsedProduct{
				ProductID: product.ID.String(),
				Quantity:  qty,
			})
		}
	}
	return result
}

// extractQtyBeforeName looks for a digit immediately before the product name in text.
func extractQtyBeforeName(text, productName string) int {
	idx := strings.Index(text, productName)
	if idx <= 0 {
		return 1
	}
	words := strings.Fields(text[:idx])
	for i := len(words) - 1; i >= 0; i-- {
		if n, err := strconv.Atoi(words[i]); err == nil && n > 0 {
			return n
		}
	}
	return 1
}

// formatPriceSGD formats price to 2 decimal places
func formatPriceSGD(price float64) string {
	return fmt.Sprintf("%.2f", price)
}

// createStripeCheckoutSession creates a Stripe invoice for the order
func (cont *TelegramContImpl) createStripeCheckoutSession(schema string, order *domains.Order, customer *domains.Customer, customerEmail string) (*string, *string, error) {
	settings, err := cont.SettingRepo.GetByGroupAndSubGroupName(cont.Db, schema, "integration", "Stripe Client")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get Stripe settings: %w", err)
	}

	stripeSecretKey := ""
	for _, s := range settings {
		if s.Name == "stripe-client-secret-key" {
			stripeSecretKey = s.Value
			break
		}
	}

	if stripeSecretKey == "" {
		return nil, nil, fmt.Errorf("Stripe secret key not configured")
	}

	stripe.Key = stripeSecretKey

	stripeCustomer, err := stripecustomer.New(&stripe.CustomerParams{
		Name:  stripe.String(customer.Name),
		Email: stripe.String(customerEmail),
		Metadata: map[string]string{
			"order_id":    strconv.Itoa(order.ID),
			"schema":      schema,
			"customer_id": strconv.Itoa(customer.ID),
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	log.Printf("[Stripe] Creating invoice for customer %s, amount %d cents", stripeCustomer.ID, int64(order.TotalPrice*100))

	invoiceParams := &stripe.InvoiceParams{
		Customer:         stripe.String(stripeCustomer.ID),
		CollectionMethod: stripe.String("send_invoice"),
		DaysUntilDue:     stripe.Int64(7),
		Description:      stripe.String(fmt.Sprintf("Order #%d Payment", order.ID)),
		Metadata: map[string]string{
			"order_id":    strconv.Itoa(order.ID),
			"schema":      schema,
			"customer_id": strconv.Itoa(customer.ID),
		},
	}

	stripeInvoice, err := stripeinvoice.New(invoiceParams)
	if err != nil {
		log.Printf("[Stripe] ERROR creating invoice: %v", err)
		return nil, nil, fmt.Errorf("failed to create Stripe invoice: %w", err)
	}
	log.Printf("[Stripe] Invoice created: ID=%s, status=%s", stripeInvoice.ID, stripeInvoice.Status)

	itemParams := &stripe.InvoiceItemParams{
		Customer:    stripe.String(stripeCustomer.ID),
		Invoice:     stripe.String(stripeInvoice.ID),
		Amount:      stripe.Int64(int64(order.TotalPrice * 100)),
		Currency:    stripe.String(string(stripe.CurrencySGD)),
		Description: stripe.String("Order Payment"),
		Metadata: map[string]string{
			"order_id": strconv.Itoa(order.ID),
		},
	}

	_, err = invoiceitem.New(itemParams)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to add invoice item: %w", err)
	}

	finalInv, err := stripeinvoice.FinalizeInvoice(stripeInvoice.ID, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to finalize invoice: %w", err)
	}

	log.Printf("[Stripe] Invoice finalized: ID=%s, amount_due=%d, hosted_url=%s",
		finalInv.ID, finalInv.AmountDue, finalInv.HostedInvoiceURL)

	return &finalInv.ID, &finalInv.HostedInvoiceURL, nil
}
