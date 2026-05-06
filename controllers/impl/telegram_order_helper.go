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
	checkoutsession "github.com/stripe/stripe-go/v81/checkout/session"
)

// ParsedProduct is the structured product selection extracted by AI.
type ParsedProduct struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// startCreateOrder starts the order creation flow
func (cont *TelegramContImpl) startCreateOrder(tgClient *helpers.TelegramClient, chatID, schema string, guest *domains.Guest, clientID uuid.UUID) {
	log.Printf("[Order] Starting order creation for guest %s", guest.ID)

	if guest.ConversationState == nil {
		guest.ConversationState = domains.JSONB{}
	}
	guest.ConversationState["state"] = "creating_order"
	guest.ConversationState["guest_phone"] = guest.Phone

	// Offer tag filter if any tags exist
	tags, _ := cont.ProductTagRepo.GetAll(cont.Db, schema)
	if len(tags) > 0 {
		tagsJSON, _ := json.Marshal(tags)
		guest.ConversationState["available_tags"] = string(tagsJSON)
		guest.ConversationState["order_step"] = "tags"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		msg := "🏷️ *Filter products by tag (optional):*\n\n"
		for i, tag := range tags {
			msg += fmt.Sprintf("*%d.* %s\n", i+1, tag.Name)
		}
		msg += "\nEnter tag number(s) separated by comma (e.g. `1` or `1,2`)\n"
		msg += "Type `all` to browse all products\n"
		msg += "Type `menu` to cancel"
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, msg)
		return
	}

	// No tags — go straight to product selection
	cont.sendProductList(tgClient, chatID, schema, guest, clientID, nil)
}

// sendProductList fetches products (filtered by tagIDs if given) and shows the selection prompt.
// tagIDs nil/empty means all products.
func (cont *TelegramContImpl) sendProductList(tgClient *helpers.TelegramClient, chatID, schema string, guest *domains.Guest, clientID uuid.UUID, tagIDs []uuid.UUID) {
	var products []domains.Product
	var total int
	if len(tagIDs) > 0 {
		products, total, _ = cont.ProductRepo.GetAllByTagIDs(cont.Db, schema, tagIDs, domains.Pagination{Page: 1, Limit: 20})
	} else {
		products, total, _ = cont.ProductRepo.GetAll(cont.Db, schema, domains.Pagination{Page: 1, Limit: 20})
	}

	if total == 0 {
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema,
			"📦 No products available for the selected tag(s).\n\nType `menu` to go back.")
		return
	}

	message := "🛒 *Please select the products you would like to order:*\n\n"
	for i, p := range products {
		line := fmt.Sprintf("*%d. %s* - $%s", i+1, p.Name, formatPriceSGD(p.Price))
		if p.IsOutOfStock {
			line += " _(SOLD OUT)_"
		} else if p.ProductQuantity > 0 {
			line += fmt.Sprintf(" _(%d left)_", p.ProductQuantity)
		}
		message += line + "\n"
		if p.Description != nil && *p.Description != "" {
			message += fmt.Sprintf("   _%s_\n", *p.Description)
		}
	}
	message += "\nJust tell me what you'd like, e.g.:\n"
	message += "• _'1 Mie Tek Tek'_\n"
	message += "• _'2 Fried Rice and 1 Iced Tea'_\n"
	message += "\nType `menu` to cancel"

	guest.ConversationState["order_step"] = "products"
	cont.GuestRepo.Update(cont.Db, schema, *guest)
	cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, message)
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
		delete(guest.ConversationState, "selected_tag_ids")
		delete(guest.ConversationState, "available_tags")
		delete(guest.ConversationState, "delivery_charge")
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
	case "tags":
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}

		// Decode available tags stored at start
		var availableTags []domains.ProductTag
		if tagsJSON, ok := guest.ConversationState["available_tags"].(string); ok {
			json.Unmarshal([]byte(tagsJSON), &availableTags)
		}

		var selectedTagIDs []string
		var selectedTagNames []string

		if !strings.EqualFold(strings.TrimSpace(text), "all") {
			for _, part := range strings.Split(text, ",") {
				num, err := strconv.Atoi(strings.TrimSpace(part))
				if err == nil && num >= 1 && num <= len(availableTags) {
					selectedTagIDs = append(selectedTagIDs, availableTags[num-1].ID.String())
					selectedTagNames = append(selectedTagNames, availableTags[num-1].Name)
				}
			}
		}

		// Store selected tag IDs for finalize and for product filtering
		if len(selectedTagIDs) > 0 {
			tagIDsJSON, _ := json.Marshal(selectedTagIDs)
			guest.ConversationState["selected_tag_ids"] = string(tagIDsJSON)
		}

		// Build uuid slice for repo filter
		var tagUUIDs []uuid.UUID
		for _, idStr := range selectedTagIDs {
			if parsed, err := uuid.Parse(idStr); err == nil {
				tagUUIDs = append(tagUUIDs, parsed)
			}
		}

		if len(selectedTagNames) > 0 {
			log.Printf("[Order] Tag filter selected: %v", selectedTagNames)
		} else {
			log.Printf("[Order] No tag filter — showing all products")
		}

		cont.sendProductList(tgClient, chatID, schema, guest, clientID, tagUUIDs)

	case "products":
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}

		// Load products — filtered by tags if user selected any
		var allProducts []domains.Product
		if tagIDsJSON, ok := guest.ConversationState["selected_tag_ids"].(string); ok && tagIDsJSON != "" {
			var tagIDStrs []string
			if err := json.Unmarshal([]byte(tagIDsJSON), &tagIDStrs); err == nil {
				var tagUUIDs []uuid.UUID
				for _, idStr := range tagIDStrs {
					if parsed, err := uuid.Parse(idStr); err == nil {
						tagUUIDs = append(tagUUIDs, parsed)
					}
				}
				if len(tagUUIDs) > 0 {
					allProducts, _, _ = cont.ProductRepo.GetAllByTagIDs(cont.Db, schema, tagUUIDs, domains.Pagination{Page: 1, Limit: 100})
				}
			}
		}
		if len(allProducts) == 0 {
			allProducts, _, _ = cont.ProductRepo.GetAll(cont.Db, schema, domains.Pagination{Page: 1, Limit: 100})
		}

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

		// Stock availability check
		var stockIssues []string
		for _, p := range parsed {
			for _, prod := range allProducts {
				if prod.ID.String() == p.ProductID {
					if prod.IsOutOfStock {
						stockIssues = append(stockIssues, fmt.Sprintf("• *%s* is sold out", prod.Name))
					} else if prod.ProductQuantity > 0 && p.Quantity > prod.ProductQuantity {
						stockIssues = append(stockIssues, fmt.Sprintf("• *%s* — only %d left (you requested %d)", prod.Name, prod.ProductQuantity, p.Quantity))
					}
					break
				}
			}
		}
		if len(stockIssues) > 0 {
			msg := "⚠️ *Some items are unavailable:*\n\n"
			for _, s := range stockIssues {
				msg += s + "\n"
			}
			msg += "\nPlease adjust your order and try again.\n\nType `menu` to cancel"
			cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, msg)
			return
		}

		// Build confirmation message
		confirmMsg := "✅ *Products to be ordered:*\n"
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
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		guest.ConversationState["postal_code"] = text

		// Calculate subtotal and delivery charge from ordered products
		var subtotal float64
		var deliveryCharge float64
		if productsParsedJSON, ok := guest.ConversationState["products_parsed"].(string); ok && productsParsedJSON != "" {
			var parsed []ParsedProduct
			if err := json.Unmarshal([]byte(productsParsedJSON), &parsed); err == nil {
				allProducts, _, _ := cont.ProductRepo.GetAll(cont.Db, schema, domains.Pagination{Page: 1, Limit: 100})
				chargeFound := false
				for _, p := range parsed {
					for _, prod := range allProducts {
						if prod.ID.String() == p.ProductID && p.Quantity > 0 {
							subtotal += prod.Price * float64(p.Quantity)
							if !chargeFound {
								settings, err := cont.DeliverySettingRepo.GetBySubGroupName(cont.Db, schema, prod.DeliveryID.String())
								if err == nil && len(settings) > 0 {
									ds := domains.ToDeliverySetting(settings)
									if len(ds) > 0 && ds[0].DeliveryType == "Delivery" && ds[0].Charge > 0 {
										deliveryCharge = ds[0].Charge
									}
								}
								chargeFound = true
							}
							break
						}
					}
				}
			}
		}

		if deliveryCharge > 0 {
			guest.ConversationState["delivery_charge"] = fmt.Sprintf("%.2f", deliveryCharge)
		}
		guest.ConversationState["order_step"] = "payment_method"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		msg := ""
		if deliveryCharge > 0 {
			msg += "🧾 *Order summary:*\n"
			msg += fmt.Sprintf("  Subtotal: $%s\n", formatPriceSGD(subtotal))
			msg += fmt.Sprintf("  Delivery: $%s\n", formatPriceSGD(deliveryCharge))
			msg += fmt.Sprintf("  *Total: $%s*\n\n", formatPriceSGD(subtotal+deliveryCharge))
		}
		msg += "💳 *Choose payment method:*\n\n"
		msg += "*1.* Cash on Delivery (COD)\n"
		msg += "*2.* Stripe (Online Payment)\n\n"
		msg += "Type `menu` to cancel"
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, msg)

	case "payment_method":
		choice := strings.TrimSpace(text)
		var paymentMethod string
		switch choice {
		case "1":
			paymentMethod = "cash_on_delivery"
		case "2":
			paymentMethod = "stripe"
		default:
			cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema,
				"⚠️ Please enter *1* for Cash on Delivery or *2* for Stripe.\n\nType `menu` to cancel")
			return
		}
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		guest.ConversationState["payment_method"] = paymentMethod
		cont.GuestRepo.Update(cont.Db, schema, *guest)
		cont.finalizeCreateOrder(tgClient, chatID, schema, guest, clientID)

	default:
		log.Printf("[Order] Unknown order_step: %s", orderStep)
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "⚠️ Invalid step. Please type 'menu' to start over.")
	}
}

// finalizeCreateOrder creates the order and saves to database
func (cont *TelegramContImpl) finalizeCreateOrder(tgClient *helpers.TelegramClient, chatID, schema string, guest *domains.Guest, clientID uuid.UUID) {
	if guest.ConversationState == nil {
		guest.ConversationState = domains.JSONB{}
	}

	productsParsedJSON, _ := guest.ConversationState["products_parsed"].(string)
	customerName, _ := guest.ConversationState["customer_name"].(string)
	customerEmail, _ := guest.ConversationState["customer_email"].(string)
	address, _ := guest.ConversationState["address"].(string)
	postalCode, _ := guest.ConversationState["postal_code"].(string)
	guestPhone, _ := guest.ConversationState["guest_phone"].(string)
	selectedPaymentMethod, _ := guest.ConversationState["payment_method"].(string)
	if selectedPaymentMethod == "" {
		selectedPaymentMethod = "stripe"
	}

	log.Printf("[Order] Finalizing order: name=%s, email=%s, phone=%s, address=%s, postal=%s, payment=%s",
		customerName, customerEmail, guestPhone, address, postalCode, selectedPaymentMethod)

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

	deliverySubGroupName := ""
	if productsParsedJSON != "" {
		var parsed []ParsedProduct
		if err := json.Unmarshal([]byte(productsParsedJSON), &parsed); err == nil {
			for _, p := range parsed {
				for _, prod := range allProducts {
					if prod.ID.String() == p.ProductID && p.Quantity > 0 {
						if deliverySubGroupName == "" {
							deliverySubGroupName = prod.DeliveryID.String()
						}
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

	// Add delivery charge to total if set
	subtotalPrice := totalPrice
	var deliveryCharge float64
	if chargeStr, ok := guest.ConversationState["delivery_charge"].(string); ok && chargeStr != "" {
		if charge, err := strconv.ParseFloat(chargeStr, 64); err == nil && charge > 0 {
			deliveryCharge = charge
			totalPrice += charge
		}
	}

	// 3. Create order — include tag filter IDs for tracking
	var tagFilterIDs []string
	if tagIDsJSON, ok := guest.ConversationState["selected_tag_ids"].(string); ok && tagIDsJSON != "" {
		json.Unmarshal([]byte(tagIDsJSON), &tagFilterIDs)
	}

	order := &domains.Order{
		CustomerID:           customer.ID,
		TotalPrice:           totalPrice,
		Status:               domains.OrderStatusPending,
		DeliverySubGroupName: deliverySubGroupName,
		StreetAddress:        address,
		PostalCode:           postalCode,
		TagFilterIDs:         tagFilterIDs,
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

	// 4b. Decrement stock for products that track quantity
	productMap := make(map[string]domains.Product)
	for _, prod := range allProducts {
		productMap[prod.ID.String()] = prod
	}
	for _, op := range orderProducts {
		prod, ok := productMap[op.ProductID]
		if !ok || prod.ProductQuantity <= 0 {
			continue
		}
		productUUID, parseErr := uuid.Parse(op.ProductID)
		if parseErr != nil {
			continue
		}
		if err := cont.ProductRepo.DecrementQuantity(tx, schema, productUUID, op.Quantity); err != nil {
			tx.Rollback()
			log.Printf("[Order] ERROR: Stock decrement failed for product %s: %v", op.ProductID, err)
			cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "❌ Insufficient stock for one or more items. Please try again.")
			return
		}
	}

	// 5. Payment: COD or Stripe
	var orderPayment *domains.OrderPayment

	if selectedPaymentMethod == "cash_on_delivery" {
		log.Printf("[Order] Payment method: Cash on Delivery")
		orderPayment = &domains.OrderPayment{
			OrderID:       order.ID,
			PaymentStatus: domains.PaymentStatusUnpaid,
			PaymentMethod: "cash_on_delivery",
			PaymentGateway: "cod",
			TotalPrice:    totalPrice,
			ExpireAt:      order.CreatedAt.Add(24 * time.Hour),
		}
	} else {
		// Stripe
		log.Printf("[Order] Creating Stripe checkout session for order ID=%d", order.ID)
		sessionID, sessionURL, stripeErr := cont.createStripeCheckoutSession(schema, order, customerEmail)
		if stripeErr != nil {
			tx.Rollback()
			log.Printf("[Order] ERROR: Failed to create Stripe checkout session: %v", stripeErr)
			cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "❌ Failed to create payment link. Please try again.")
			return
		}
		log.Printf("[Order] Stripe checkout session created: ID=%s", *sessionID)

		stripePaymentStatus := "open"
		orderPayment = &domains.OrderPayment{
			OrderID:              order.ID,
			PaymentStatus:        domains.PaymentStatusUnpaid,
			PaymentMethod:        "stripe",
			PaymentGateway:       "stripe",
			TotalPrice:           totalPrice,
			ExpireAt:             order.CreatedAt.Add(15 * time.Minute),
			PaymentSessionID:     sessionID,
			PaymentSessionURL:    sessionURL,
			PaymentGatewayStatus: &stripePaymentStatus,
			PaymentInvoiceID:     sessionID,
		}
	}

	// 6. Save order payment
	existingPayment, _ := cont.OrderPaymentRepo.GetByOrderID(tx, schema, order.ID)
	if existingPayment != nil && existingPayment.ID != uuid.Nil {
		tx.Table(schema+".order_payments").Where("id = ?", existingPayment.ID).
			Updates(map[string]interface{}{
				"payment_method":         orderPayment.PaymentMethod,
				"payment_session_id":     orderPayment.PaymentSessionID,
				"payment_session_url":    orderPayment.PaymentSessionURL,
				"payment_gateway_status": orderPayment.PaymentGatewayStatus,
				"payment_invoice_id":     orderPayment.PaymentInvoiceID,
			})
	} else {
		if _, err = cont.OrderPaymentRepo.Create(tx, schema, *orderPayment); err != nil {
			tx.Rollback()
			log.Printf("[Order] ERROR: Failed to create order payment: %v", err)
			cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "❌ Error creating payment. Please try again.")
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		log.Printf("[Order] ERROR: Failed to commit transaction: %v", err)
		cont.sendBotMessage(tgClient, clientID, guest.ID, guest.Name, chatID, schema, "❌ Error finalizing order. Please try again.")
		return
	}

	log.Printf("[Order] ✅ Order created successfully: ID=%d, payment=%s", order.ID, selectedPaymentMethod)

	// 7. Send success message
	summary := "🎉 *Order Created!*\n\n"
	summary += "✅ *Order details:*\n"
	summary += fmt.Sprintf("- Order ID: #%d\n", order.ID)
	summary += fmt.Sprintf("- Items: %s\n", productsSummary)
	summary += fmt.Sprintf("- Name: %s\n", customerName)
	summary += fmt.Sprintf("- Phone: %s%s\n", phoneCountryCode, phoneNumber)
	summary += fmt.Sprintf("- Address: %s\n", address)
	summary += fmt.Sprintf("- Postal code: %s\n", postalCode)
	if deliveryCharge > 0 {
		summary += fmt.Sprintf("- Subtotal: $%s\n", formatPriceSGD(subtotalPrice))
		summary += fmt.Sprintf("- Delivery: $%s\n", formatPriceSGD(deliveryCharge))
	}
	summary += fmt.Sprintf("- Total: $%s\n", formatPriceSGD(totalPrice))

	if selectedPaymentMethod == "cash_on_delivery" {
		summary += "\n💵 *Payment: Cash on Delivery*\n"
		summary += fmt.Sprintf("Please prepare *$%s* cash upon delivery.\n\n", formatPriceSGD(totalPrice))
	} else {
		summary += "\n💳 *Pay Now:*\n"
		summary += fmt.Sprintf("%s\n\n", *orderPayment.PaymentSessionURL)
		summary += "⏰ *Payment link expires in 15 minutes!*\n\n"
	}
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
	delete(guest.ConversationState, "payment_method")
	delete(guest.ConversationState, "selected_tag_ids")
	delete(guest.ConversationState, "available_tags")
	delete(guest.ConversationState, "delivery_charge")
	cont.GuestRepo.Update(cont.Db, schema, *guest)
}

// parseProductsWithAI calls OpenAI to extract product selections from any natural language text.
// The AI understands any language and phrasing — "1 fried rice", "2 portions of noodles and an iced tea", etc.
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
User: "i want 1 fried rice" → [{"product_id":"<uuid>","quantity":1}]
User: "2 fried rice and 1 iced tea" → [{"product_id":"<uuid-fried-rice>","quantity":2},{"product_id":"<uuid-iced-tea>","quantity":1}]
User: "give me 3 of the first one" → [{"product_id":"<uuid-first-product>","quantity":3}]`

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

// createStripeCheckoutSession creates a Stripe Checkout Session for the order.
// Returns (sessionID, sessionURL, error). Does NOT require customer email.
func (cont *TelegramContImpl) createStripeCheckoutSession(schema string, order *domains.Order, customerEmail string) (*string, *string, error) {
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

	successURL := os.Getenv("STRIPE_SUCCESS_URL")
	cancelURL := os.Getenv("STRIPE_CANCEL_URL")
	if successURL == "" {
		successURL = "https://example.com/payment/success"
	}
	if cancelURL == "" {
		cancelURL = "https://example.com/payment/cancel"
	}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(string(stripe.CurrencySGD)),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(fmt.Sprintf("Order #%d", order.ID)),
					},
					UnitAmount: stripe.Int64(int64(order.TotalPrice * 100)),
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		Metadata: map[string]string{
			"order_id": strconv.Itoa(order.ID),
			"schema":   schema,
		},
		ExpiresAt: stripe.Int64(time.Now().Add(30 * time.Minute).Unix()),
	}

	if customerEmail != "" {
		params.CustomerEmail = stripe.String(customerEmail)
	}

	session, err := checkoutsession.New(params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Stripe checkout session: %w", err)
	}

	log.Printf("[Stripe] Checkout session created: ID=%s", session.ID)
	return &session.ID, &session.URL, nil
}
