package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	checkoutsession "github.com/stripe/stripe-go/v81/checkout/session"
)

// waStartCreateOrder starts the order creation flow for WhatsApp
func (cont *WhatsAppContImpl) waStartCreateOrder(waClient helpers.WhatsAppSender, chatID, schema string, guest *domains.Guest) {
	log.Printf("[WhatsApp/Order] starting order creation for guest %s", guest.ID)

	if guest.ConversationState == nil {
		guest.ConversationState = domains.JSONB{}
	}
	guest.ConversationState["state"] = "creating_order"
	guest.ConversationState["guest_phone"] = guest.Phone

	// LID accounts don't expose a real phone number — ask the customer to provide it
	if guest.Phone == "" {
		guest.ConversationState["order_step"] = "phone"
		cont.GuestRepo.Update(cont.Db, schema, *guest)
		cont.sendWABotMessage(waClient, uuid.Nil, guest.ID, guest.Name, chatID, schema,
			"📱 Masukkan nomor WhatsApp kamu (dengan kode negara):\n\nContoh: +6281234567890\n\nKetik 'menu' untuk batal")
		return
	}

	// Offer tag filter if any tags exist
	tags, _ := cont.ProductTagRepo.GetAll(cont.Db, schema)
	if len(tags) > 0 {
		tagsJSON, _ := json.Marshal(tags)
		guest.ConversationState["available_tags"] = string(tagsJSON)
		guest.ConversationState["order_step"] = "tags"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		msg := "🏷️ Filter produk berdasarkan tag (opsional):\n\n"
		for i, tag := range tags {
			msg += fmt.Sprintf("%d. %s\n", i+1, tag.Name)
		}
		msg += "\nMasukkan nomor tag, pisahkan dengan koma (contoh: 1 atau 1,2)\n"
		msg += "Ketik 'all' untuk lihat semua produk\n"
		msg += "Ketik 'menu' untuk batal"
		cont.sendWABotMessage(waClient, uuid.Nil, guest.ID, guest.Name, chatID, schema, msg)
		return
	}

	// No tags — go straight to product selection
	cont.waSendProductList(waClient, chatID, schema, guest, uuid.Nil, nil)
}

// waSendProductList fetches products (filtered by tagIDs if given) and shows the selection prompt.
func (cont *WhatsAppContImpl) waSendProductList(waClient helpers.WhatsAppSender, chatID, schema string, guest *domains.Guest, clientID uuid.UUID, tagIDs []uuid.UUID) {
	var products []domains.Product
	var total int
	if len(tagIDs) > 0 {
		products, total, _ = cont.ProductRepo.GetAllByTagIDs(cont.Db, schema, tagIDs, domains.Pagination{Page: 1, Limit: 20})
	} else {
		products, total, _ = cont.ProductRepo.GetAll(cont.Db, schema, domains.Pagination{Page: 1, Limit: 20})
	}

	if total == 0 {
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"📦 Tidak ada produk tersedia untuk tag yang dipilih.\n\nKetik 'menu' untuk kembali.")
		return
	}

	message := "🛒 Pilih produk yang ingin dipesan:\n\n"
	for i, p := range products {
		line := fmt.Sprintf("%d. %s - $%s", i+1, p.Name, formatPriceSGD(p.Price))
		if p.IsOutOfStock {
			line += " (HABIS)"
		} else if p.ProductQuantity > 0 {
			line += fmt.Sprintf(" (sisa %d)", p.ProductQuantity)
		}
		message += line + "\n"
		if p.Description != nil && *p.Description != "" {
			message += fmt.Sprintf("   %s\n", *p.Description)
		}
	}
	message += "\nSebutkan produk dan jumlahnya, contoh:\n"
	message += "• '1 Mie Tek Tek'\n"
	message += "• '2 Nasi Goreng dan 1 Es Teh'\n"
	message += "\nKetik 'menu' untuk batal"

	guest.ConversationState["order_step"] = "products"
	cont.GuestRepo.Update(cont.Db, schema, *guest)
	cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema, message)
}

// waContinueCreateOrder continues the order creation flow
func (cont *WhatsAppContImpl) waContinueCreateOrder(waClient helpers.WhatsAppSender, chatID, schema string, guest *domains.Guest, text string, clientID uuid.UUID) {
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
		delete(guest.ConversationState, "payment_method")
		delete(guest.ConversationState, "selected_tag_ids")
		delete(guest.ConversationState, "available_tags")
		delete(guest.ConversationState, "delivery_charge")
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
			"⚠️ Sesi pesanan kadaluwarsa. Ketik 'menu' untuk mulai ulang.")
		return
	}

	switch orderStep {
	case "phone":
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		phone := strings.TrimSpace(text)
		if len(phone) < 8 {
			cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
				"⚠️ Masukkan nomor HP yang valid (contoh: +6281234567890).\n\nKetik 'menu' untuk batal")
			return
		}
		guest.ConversationState["guest_phone"] = phone
		tags, _ := cont.ProductTagRepo.GetAll(cont.Db, schema)
		if len(tags) > 0 {
			tagsJSON, _ := json.Marshal(tags)
			guest.ConversationState["available_tags"] = string(tagsJSON)
			guest.ConversationState["order_step"] = "tags"
			cont.GuestRepo.Update(cont.Db, schema, *guest)
			msg := "🏷️ Filter produk berdasarkan tag (opsional):\n\n"
			for i, tag := range tags {
				msg += fmt.Sprintf("%d. %s\n", i+1, tag.Name)
			}
			msg += "\nMasukkan nomor tag, pisahkan dengan koma (contoh: 1 atau 1,2)\n"
			msg += "Ketik 'all' untuk lihat semua produk\n"
			msg += "Ketik 'menu' untuk batal"
			cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema, msg)
		} else {
			cont.waSendProductList(waClient, chatID, schema, guest, clientID, nil)
		}

	case "tags":
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}

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

		if len(selectedTagIDs) > 0 {
			tagIDsJSON, _ := json.Marshal(selectedTagIDs)
			guest.ConversationState["selected_tag_ids"] = string(tagIDsJSON)
		}

		var tagUUIDs []uuid.UUID
		for _, idStr := range selectedTagIDs {
			if parsed, err := uuid.Parse(idStr); err == nil {
				tagUUIDs = append(tagUUIDs, parsed)
			}
		}

		if len(selectedTagNames) > 0 {
			log.Printf("[WhatsApp/Order] Tag filter selected: %v", selectedTagNames)
		}

		cont.waSendProductList(waClient, chatID, schema, guest, clientID, tagUUIDs)

	case "products":
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}

		// Load products filtered by tags if selected
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

		parsed, err := parseProductsWithAI(text, allProducts)
		if err != nil {
			log.Printf("[WhatsApp/Order] AI parse failed (%v), falling back to text match", err)
			parsed = parseProductsFallback(text, allProducts)
		}

		if len(parsed) == 0 {
			msg := "⚠️ Saya tidak bisa mengenali pilihan produkmu.\n\n"
			msg += "Sebutkan nama produk dan jumlahnya, contoh:\n"
			msg += "• '1 Mie Tek Tek'\n"
			msg += "• '2 Nasi Goreng dan 1 Es Teh'\n\n"
			msg += "Ketik 'menu' untuk batal"
			cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema, msg)
			return
		}

		// Stock availability check
		var stockIssues []string
		for _, p := range parsed {
			for _, prod := range allProducts {
				if prod.ID.String() == p.ProductID {
					if prod.IsOutOfStock {
						stockIssues = append(stockIssues, fmt.Sprintf("• %s sudah habis", prod.Name))
					} else if prod.ProductQuantity > 0 && p.Quantity > prod.ProductQuantity {
						stockIssues = append(stockIssues, fmt.Sprintf("• %s — hanya tersisa %d (kamu minta %d)", prod.Name, prod.ProductQuantity, p.Quantity))
					}
					break
				}
			}
		}
		if len(stockIssues) > 0 {
			msg := "⚠️ Beberapa item tidak tersedia:\n\n"
			for _, s := range stockIssues {
				msg += s + "\n"
			}
			msg += "\nSesuaikan pesananmu dan coba lagi.\n\nKetik 'menu' untuk batal"
			cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema, msg)
			return
		}

		confirmMsg := "✅ Produk yang dipesan:\n"
		for _, p := range parsed {
			for _, prod := range allProducts {
				if prod.ID.String() == p.ProductID {
					confirmMsg += fmt.Sprintf("• %dx %s - $%s\n", p.Quantity, prod.Name, formatPriceSGD(prod.Price*float64(p.Quantity)))
					break
				}
			}
		}

		parsedJSON, _ := json.Marshal(parsed)
		guest.ConversationState["products_parsed"] = string(parsedJSON)
		guest.ConversationState["order_step"] = "name"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			confirmMsg+"\nSiapa nama lengkap kamu?\n\nKetik 'menu' untuk batal")

	case "name":
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		guest.ConversationState["customer_name"] = text
		guest.ConversationState["order_step"] = "email"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"✅ Nama tersimpan!\n\nAlamat email kamu? (untuk pengiriman invoice)\n\nContoh: nama@gmail.com\n\nKetik 'menu' untuk batal")

	case "email":
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		guest.ConversationState["customer_email"] = text
		guest.ConversationState["order_step"] = "address"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"✅ Email tersimpan!\n\nAlamat pengiriman? (Jalan, nomor, gedung, dll.)\n\nKetik 'menu' untuk batal")

	case "address":
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		guest.ConversationState["address"] = text
		guest.ConversationState["order_step"] = "postal_code"
		cont.GuestRepo.Update(cont.Db, schema, *guest)

		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"✅ Alamat tersimpan!\n\nKode pos?\n\nKetik 'menu' untuk batal")

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
			msg += "🧾 Ringkasan pesanan:\n"
			msg += fmt.Sprintf("  Subtotal: $%s\n", formatPriceSGD(subtotal))
			msg += fmt.Sprintf("  Ongkir: $%s\n", formatPriceSGD(deliveryCharge))
			msg += fmt.Sprintf("  Total: $%s\n\n", formatPriceSGD(subtotal+deliveryCharge))
		}
		msg += "💳 Pilih metode pembayaran:\n\n"
		msg += "1. Bayar di tempat (COD)\n"
		msg += "2. Stripe (Pembayaran Online)\n\n"
		msg += "Ketik 'menu' untuk batal"
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema, msg)

	case "payment_method":
		choice := strings.TrimSpace(text)
		var paymentMethod string
		switch choice {
		case "1":
			paymentMethod = "cash_on_delivery"
		case "2":
			paymentMethod = "stripe"
		default:
			cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
				"⚠️ Ketik 1 untuk Bayar di Tempat atau 2 untuk Stripe.\n\nKetik 'menu' untuk batal")
			return
		}
		if guest.ConversationState == nil {
			guest.ConversationState = domains.JSONB{}
		}
		guest.ConversationState["payment_method"] = paymentMethod
		cont.GuestRepo.Update(cont.Db, schema, *guest)
		cont.waFinalizeCreateOrder(waClient, chatID, schema, guest, clientID)

	default:
		log.Printf("[WhatsApp/Order] unknown order_step: %s", orderStep)
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"⚠️ Sesi tidak valid. Ketik 'menu' untuk mulai ulang.")
	}
}

// waFinalizeCreateOrder creates the order and saves it to the database
func (cont *WhatsAppContImpl) waFinalizeCreateOrder(waClient helpers.WhatsAppSender, chatID, schema string, guest *domains.Guest, clientID uuid.UUID) {
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

	log.Printf("[WhatsApp/Order] finalizing: name=%s, email=%s, phone=%s, address=%s, postal=%s, payment=%s",
		customerName, customerEmail, guestPhone, address, postalCode, selectedPaymentMethod)

	phoneCountryCode := ""
	phoneNumber := guestPhone
	if guestPhone != "" {
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, guestPhone)
		digits = strings.TrimLeft(digits, "0")
		if len(digits) > 2 {
			phoneCountryCode = digits[:2]
			phoneNumber = digits[2:]
		}
	}

	tx := cont.Db.Begin()
	if tx.Error != nil {
		log.Printf("[WhatsApp/Order] ERROR: Failed to start transaction: %v", tx.Error)
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"❌ Gagal membuat pesanan. Silakan coba lagi.")
		return
	}

	toPtr := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}

	// Find or create customer
	var customer *domains.Customer
	if phoneCountryCode != "" {
		customer, _ = cont.CustomerRepo.GetByConcatPhone(tx, schema, phoneCountryCode+phoneNumber)
	}
	if customer == nil && guest.Username != "" {
		customer, _ = cont.CustomerRepo.GetByUsername(tx, schema, guest.Username)
	}
	if customer == nil {
		newCust := domains.Customer{
			Name:             customerName,
			Username:         toPtr(guest.Username),
			PhoneCountryCode: toPtr(phoneCountryCode),
			PhoneNumber:      toPtr(phoneNumber),
			Address:          toPtr(address),
			PostalCode:       toPtr(postalCode),
			AccountType:      "Whatsapp",
		}
		var createErr error
		customer, createErr = cont.CustomerRepo.Create(tx, schema, newCust)
		if createErr != nil {
			tx.Rollback()
			log.Printf("[WhatsApp/Order] ERROR: Failed to create customer: %v", createErr)
			cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
				"❌ Gagal membuat data pelanggan. Silakan coba lagi.")
			return
		}
	}

	// Build order products
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
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"❌ Tidak ada produk valid dalam pesanan. Silakan coba lagi.")
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

	// Create order — include tag filter IDs for tracking
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

	order, err := cont.OrderRepo.Create(tx, schema, *order)
	if err != nil {
		tx.Rollback()
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"❌ Gagal membuat pesanan. Silakan coba lagi.")
		return
	}

	// Create order products
	for i := range orderProducts {
		orderProducts[i].OrderID = order.ID
	}
	if err := cont.OrderRepo.CreateOrderProducts(tx, schema, orderProducts); err != nil {
		tx.Rollback()
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"❌ Gagal menyimpan produk pesanan. Silakan coba lagi.")
		return
	}

	// Decrement stock for products that track quantity
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
			log.Printf("[WhatsApp/Order] ERROR: Stock decrement failed for product %s: %v", op.ProductID, err)
			cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
				"❌ Stok tidak cukup untuk satu atau lebih item. Silakan coba lagi.")
			return
		}
	}

	// Payment: COD or Stripe
	var orderPayment *domains.OrderPayment

	if selectedPaymentMethod == "cash_on_delivery" {
		log.Printf("[WhatsApp/Order] Payment method: Cash on Delivery")
		orderPayment = &domains.OrderPayment{
			OrderID:        order.ID,
			PaymentStatus:  domains.PaymentStatusUnpaid,
			PaymentMethod:  "cash_on_delivery",
			PaymentGateway: "cod",
			TotalPrice:     totalPrice,
			ExpireAt:       order.CreatedAt.Add(24 * time.Hour),
		}
	} else {
		sessionID, sessionURL, stripeErr := cont.waCreateStripeCheckout(schema, order, customerEmail)
		if stripeErr != nil {
			tx.Rollback()
			log.Printf("[WhatsApp/Order] Stripe checkout error: %v", stripeErr)
			cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
				"❌ Gagal membuat link pembayaran. Silakan coba lagi.")
			return
		}
		stripeStatus := "open"
		orderPayment = &domains.OrderPayment{
			OrderID:              order.ID,
			PaymentStatus:        domains.PaymentStatusUnpaid,
			PaymentMethod:        "stripe",
			PaymentGateway:       "stripe",
			TotalPrice:           totalPrice,
			ExpireAt:             order.CreatedAt.Add(15 * time.Minute),
			PaymentSessionID:     sessionID,
			PaymentSessionURL:    sessionURL,
			PaymentGatewayStatus: &stripeStatus,
			PaymentInvoiceID:     sessionID,
		}
	}

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
		if _, err := cont.OrderPaymentRepo.Create(tx, schema, *orderPayment); err != nil {
			tx.Rollback()
			cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
				"❌ Gagal membuat pembayaran. Silakan coba lagi.")
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		cont.sendWABotMessage(waClient, clientID, guest.ID, guest.Name, chatID, schema,
			"❌ Gagal menyelesaikan pesanan. Silakan coba lagi.")
		return
	}

	log.Printf("[WhatsApp/Order] ✅ Order created: ID=%d, payment=%s", order.ID, selectedPaymentMethod)

	// Fire order notification (WA + email) in background
	go func() {
		// Prefer whatsmeow (no 24h session restriction). Fall back to Cloud API.
		var notifSender helpers.WhatsAppSender
		if cont.WhatsmeowHub != nil {
			notifSender = cont.WhatsmeowHub.GetSender(schema)
		}
		if notifSender == nil {
			waSettings, err := cont.SettingRepo.GetByGroupAndSubGroupName(cont.Db, schema, "integration", "WhatsApp")
			if err == nil {
				phoneNumberID, accessToken := "", ""
				for _, s := range waSettings {
					switch s.Name {
					case "whatsapp-phone-number-id":
						phoneNumberID = s.Value
					case "whatsapp-access-token":
						accessToken = s.Value
					}
				}
				if phoneNumberID != "" && accessToken != "" {
					notifSender = helpers.NewWhatsAppClient(phoneNumberID, accessToken)
				}
			}
		}
		helpers.SendOrderNotification(cont.Db, schema, fmt.Sprintf("%d", order.ID), customerName, totalPrice, notifSender)
	}()

	summary := "🎉 Pesanan Berhasil Dibuat!\n\n"
	summary += "✅ Detail pesanan:\n"
	summary += fmt.Sprintf("- Nomor Order: #%d\n", order.ID)
	summary += fmt.Sprintf("- Produk: %s\n", productsSummary)
	summary += fmt.Sprintf("- Nama: %s\n", customerName)
	summary += fmt.Sprintf("- Phone: %s\n", guestPhone)
	summary += fmt.Sprintf("- Alamat: %s\n", address)
	summary += fmt.Sprintf("- Kode pos: %s\n", postalCode)
	if deliveryCharge > 0 {
		summary += fmt.Sprintf("- Subtotal: $%s\n", formatPriceSGD(subtotalPrice))
		summary += fmt.Sprintf("- Ongkir: $%s\n", formatPriceSGD(deliveryCharge))
	}
	summary += fmt.Sprintf("- Total: $%s\n", formatPriceSGD(totalPrice))

	if selectedPaymentMethod == "cash_on_delivery" {
		summary += "\n💵 Pembayaran: Bayar di Tempat (COD)\n"
		summary += fmt.Sprintf("Siapkan uang tunai $%s saat pengiriman tiba.\n\n", formatPriceSGD(totalPrice))
	} else {
		summary += "\n💳 Bayar Sekarang:\n"
		summary += fmt.Sprintf("%s\n\n", *orderPayment.PaymentSessionURL)
		summary += "⏰ Link pembayaran berlaku 15 menit!\n\n"
	}
	summary += "Ketik 'menu' untuk kembali ke menu utama."

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
	delete(guest.ConversationState, "payment_method")
	delete(guest.ConversationState, "selected_tag_ids")
	delete(guest.ConversationState, "available_tags")
	delete(guest.ConversationState, "delivery_charge")
	cont.GuestRepo.Update(cont.Db, schema, *guest)
}

// waCreateStripeCheckout creates a Stripe Checkout Session for a WhatsApp order.
// Returns (sessionID, sessionURL, error). Does NOT require customer email.
func (cont *WhatsAppContImpl) waCreateStripeCheckout(schema string, order *domains.Order, customerEmail string) (*string, *string, error) {
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

	log.Printf("[WhatsApp/Stripe] Checkout session created: ID=%s", session.ID)
	return &session.ID, &session.URL, nil
}
