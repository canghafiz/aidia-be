package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	stripecustomer "github.com/stripe/stripe-go/v81/customer"
	stripeinvoice "github.com/stripe/stripe-go/v81/invoice"
	"github.com/stripe/stripe-go/v81/invoiceitem"
	"strconv"
)

// waStartCreateOrder starts the order creation flow for WhatsApp
func (cont *WhatsAppContImpl) waStartCreateOrder(waClient *helpers.WhatsAppClient, chatID, schema string, guest *domains.Guest) {
	log.Printf("[WhatsApp/Order] starting order creation for guest %s", guest.ID)

	products, total, _ := cont.ProductRepo.GetAll(cont.Db, schema, domains.Pagination{Page: 1, Limit: 20})
	if total == 0 {
		cont.sendWABotMessage(waClient, uuid.Nil, guest.ID, guest.Name, chatID, schema,
			"📦 No products available at the moment.\n\nType 'menu' to go back to main menu.")
		return
	}

	message := "🛒 Please select products you'd like to order:\n\n"
	for i, p := range products {
		message += fmt.Sprintf("%d. %s - $%s\n", i+1, p.Name, formatPriceSGD(p.Price))
		if p.Description != nil && *p.Description != "" {
			message += fmt.Sprintf("   %s\n", *p.Description)
		}
	}
	message += "\nJust tell me what you'd like, e.g.:\n"
	message += "• '1 Mie Tek Tek'\n"
	message += "• '2 Fried Rice and 1 Iced Tea'\n"
	message += "\nType 'menu' to cancel"

	cont.sendWABotMessage(waClient, uuid.Nil, guest.ID, guest.Name, chatID, schema, message)

	if guest.ConversationState == nil {
		guest.ConversationState = domains.JSONB{}
	}
	guest.ConversationState["state"] = "creating_order"
	guest.ConversationState["order_step"] = "products"
	guest.ConversationState["guest_phone"] = guest.Phone
	cont.GuestRepo.Update(cont.Db, schema, *guest)
}

// waContinueCreateOrder continues the order creation flow
func (cont *WhatsAppContImpl) waContinueCreateOrder(waClient *helpers.WhatsAppClient, chatID, schema string, guest *domains.Guest, text string, clientID uuid.UUID) {
	if strings.EqualFold(text, "menu") {
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
		cont.waShowMenu(waClient, chatID, schema, guest, clientID)
		return
	}

	orderStep := ""
	if guest.ConversationState != nil {
		if s, ok := guest.ConversationState["order_step"].(string); ok {
			orderStep = s
		}
	}

	log.Printf("[WhatsApp/Order] step=%s input=%s", orderStep, text)

	if orderStep == "" {
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"⚠️ Order session expired. Please type 'menu' to start over.")
		return
	}

	switch orderStep {
	case "products":
		allProducts, _, _ := cont.ProductRepo.GetAll(cont.Db, schema, domains.Pagination{Page: 1, Limit: 100})

		parsed, err := parseProductsWithAI(text, allProducts)
		if err != nil {
			log.Printf("[WhatsApp/Order] AI parse failed (%v), falling back to text match", err)
			parsed = parseProductsFallback(text, allProducts)
		}

		if len(parsed) == 0 {
			msg := "⚠️ I couldn't understand your product selection.\n\n"
			msg += "Please mention the product name and quantity, e.g.:\n"
			msg += "• '1 Mie Tek Tek'\n"
			msg += "• '2 Fried Rice and 1 Iced Tea'\n\n"
			msg += "Type 'menu' to cancel"
			cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema, msg)
			return
		}

		confirmMsg := "✅ Products to order:\n"
		for _, p := range parsed {
			for _, prod := range allProducts {
				if prod.ID.String() == p.ProductID {
					confirmMsg += fmt.Sprintf("• %dx %s - $%s\n", p.Quantity, prod.Name, formatPriceSGD(prod.Price*float64(p.Quantity)))
					break
				}
			}
		}

		parsedJSON, _ := json.Marshal(parsed)
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		guest.ConversationState["products_parsed"] = string(parsedJSON)
		guest.ConversationState["order_step"] = "name"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			confirmMsg+"\nWhat is your full name?\n\nType 'menu' to cancel")

	case "name":
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		guest.ConversationState["customer_name"] = text
		guest.ConversationState["order_step"] = "email"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"✅ Name saved!\n\nEmail address? (for invoice delivery)\n\nExample: test@example.com\n\nType 'menu' to cancel")

	case "email":
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		guest.ConversationState["customer_email"] = text
		guest.ConversationState["order_step"] = "address"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"✅ Email saved!\n\nDelivery address? (Street, building, etc.)\n\nType 'menu' to cancel")

	case "address":
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		guest.ConversationState["address"] = text
		guest.ConversationState["order_step"] = "postal_code"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"✅ Address saved!\n\nPostal code?\n\nType 'menu' to cancel")

	case "postal_code":
		cont.waFinalizeCreateOrder(waClient, chatID, schema, guest, text, clientID)

	default:
		log.Printf("[WhatsApp/Order] unknown order_step: %s", orderStep)
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"⚠️ Invalid step. Please type 'menu' to start over.")
	}
}

// waFinalizeCreateOrder creates the order and saves it to the database
func (cont *WhatsAppContImpl) waFinalizeCreateOrder(waClient *helpers.WhatsAppClient, chatID, schema string, guest *domains.Guest, postalCode string, clientID uuid.UUID) {
	if guest.ConversationState == nil {
		guest.ConversationState = domains.JSONB{}
	}

	productsParsedJSON, _ := guest.ConversationState["products_parsed"].(string)
	customerName, _ := guest.ConversationState["customer_name"].(string)
	customerEmail, _ := guest.ConversationState["customer_email"].(string)
	address, _ := guest.ConversationState["address"].(string)
	guestPhone, _ := guest.ConversationState["guest_phone"].(string)

	log.Printf("[WhatsApp/Order] finalizing: name=%s, email=%s, phone=%s, address=%s, postal=%s",
		customerName, customerEmail, guestPhone, address, postalCode)

	phoneCountryCode := "+62"
	phoneNumber := guestPhone
	if len(guestPhone) > 3 && guestPhone[0] == '+' {
		phoneCountryCode = guestPhone[:3]
		phoneNumber = guestPhone[3:]
	}

	tx := cont.Db.Begin()
	if tx.Error != nil {
		log.Printf("[WhatsApp/Order] ERROR: Failed to start transaction: %v", tx.Error)
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"❌ Error creating order. Please try again.")
		return
	}

	// Find or create customer
	customer, err := cont.CustomerRepo.GetByPhone(tx, schema, phoneCountryCode, phoneNumber)
	if err != nil {
		customer = &domains.Customer{
			Name:             customerName,
			PhoneCountryCode: &phoneCountryCode,
			PhoneNumber:      &phoneNumber,
			AccountType:      "WhatsApp",
		}
		customer, err = cont.CustomerRepo.Create(tx, schema, *customer)
		if err != nil {
			tx.Rollback()
			log.Printf("[WhatsApp/Order] ERROR: Failed to create customer: %v", err)
			cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
				"❌ Error creating customer. Please try again.")
			return
		}
	}

	// Build order products
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
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"❌ No valid products in order. Please try again.")
		return
	}

	// Create order
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
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"❌ Error creating order. Please try again.")
		return
	}

	// Create order products
	for i := range orderProducts {
		orderProducts[i].OrderID = order.ID
	}
	if err := cont.OrderRepo.CreateOrderProducts(tx, schema, orderProducts); err != nil {
		tx.Rollback()
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"❌ Error creating order products. Please try again.")
		return
	}

	// Create Stripe invoice
	stripeInvoiceID, stripeInvoiceURL, err := cont.waCreateStripeInvoice(schema, order, customer, customerEmail)
	if err != nil {
		log.Printf("[WhatsApp/Order] Stripe error: %v", err)
		stripeInvoiceID = nil
		stripeInvoiceURL = nil
	}

	// Update or create order payment
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
		if _, err := cont.OrderPaymentRepo.Create(tx, schema, *orderPayment); err != nil {
			tx.Rollback()
			cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
				"❌ Error creating payment. Please try again.")
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"❌ Error finalizing order. Please try again.")
		return
	}

	log.Printf("[WhatsApp/Order] ✅ Order created: ID=%d", order.ID)

	paymentLink := "Invoice will be sent separately"
	if stripeInvoiceURL != nil && *stripeInvoiceURL != "" {
		paymentLink = *stripeInvoiceURL
	} else if stripeInvoiceID != nil {
		paymentLink = fmt.Sprintf("https://invoice.stripe.com/i/%s", *stripeInvoiceID)
	}

	summary := "🎉 Order Created!\n\n"
	summary += "✅ Order details:\n"
	summary += fmt.Sprintf("- Order ID: #%d\n", order.ID)
	summary += fmt.Sprintf("- Items: %s\n", productsSummary)
	summary += fmt.Sprintf("- Name: %s\n", customerName)
	summary += fmt.Sprintf("- Phone: %s%s\n", phoneCountryCode, phoneNumber)
	summary += fmt.Sprintf("- Address: %s\n", address)
	summary += fmt.Sprintf("- Postal code: %s\n", postalCode)
	summary += fmt.Sprintf("- Total: $%s\n", formatPriceSGD(totalPrice))
	summary += "\n💳 Pay Now:\n"
	summary += fmt.Sprintf("%s\n\n", paymentLink)
	summary += "⏰ Order expires in 15 minutes!\n\n"
	summary += "Type 'menu' to go back to the main menu."

	cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema, summary)

	// Reset state
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

// waCreateStripeInvoice creates a Stripe invoice for a WhatsApp order
func (cont *WhatsAppContImpl) waCreateStripeInvoice(schema string, order *domains.Order, customer *domains.Customer, customerEmail string) (*string, *string, error) {
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
		return nil, nil, fmt.Errorf("failed to create Stripe invoice: %w", err)
	}

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

	if _, err = invoiceitem.New(itemParams); err != nil {
		return nil, nil, fmt.Errorf("failed to add invoice item: %w", err)
	}

	finalInv, err := stripeinvoice.FinalizeInvoice(stripeInvoice.ID, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to finalize invoice: %w", err)
	}

	log.Printf("[WhatsApp/Stripe] Invoice finalized: ID=%s", finalInv.ID)
	return &finalInv.ID, &finalInv.HostedInvoiceURL, nil
}
