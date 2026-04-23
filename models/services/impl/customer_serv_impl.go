package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	req "backend/models/requests/customer"
	res "backend/models/responses/customer"
	"backend/models/responses/pagination"
	"fmt"
	"log"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CustomerServImpl struct {
	Db                     *gorm.DB
	Validator              *validator.Validate
	UserRepo               repositories.UsersRepo
	CustomerRepo           repositories.CustomerRepo
	GuestRepo              repositories.GuestRepo
	GuestMessageRepo       repositories.GuestMessageRepo
	SettingRepo            repositories.SettingRepo
	WhatsAppConnectionRepo repositories.WhatsAppConnectionRepo
	JwtKey                 string
}

func normalizeTelegramUsername(raw string) string {
	return strings.TrimPrefix(strings.TrimSpace(raw), "@")
}

func NewCustomerServImpl(
	db *gorm.DB,
	validator *validator.Validate,
	userRepo repositories.UsersRepo,
	customerRepo repositories.CustomerRepo,
	guestRepo repositories.GuestRepo,
	guestMessageRepo repositories.GuestMessageRepo,
	settingRepo repositories.SettingRepo,
	whatsAppConnectionRepo repositories.WhatsAppConnectionRepo,
	jwtKey string,
) *CustomerServImpl {
	return &CustomerServImpl{
		Db:                     db,
		Validator:              validator,
		UserRepo:               userRepo,
		CustomerRepo:           customerRepo,
		GuestRepo:              guestRepo,
		GuestMessageRepo:       guestMessageRepo,
		SettingRepo:            settingRepo,
		WhatsAppConnectionRepo: whatsAppConnectionRepo,
		JwtKey:                 jwtKey,
	}
}

func (serv *CustomerServImpl) getSchema(clientID uuid.UUID) (string, error) {
	return helpers.GetSchema(serv.Db, serv.UserRepo, clientID)
}

func (serv *CustomerServImpl) checkClientRole(clientID uuid.UUID) error {
	role, err := serv.UserRepo.GetUserRole(serv.Db, clientID)
	if err != nil {
		return fmt.Errorf("failed to get user role")
	}
	if role != "Client" {
		return fmt.Errorf("access denied")
	}
	return nil
}

func (serv *CustomerServImpl) checkRole(accessToken string) error {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Admin", "Client"})
	if err != nil || !ok {
		return fmt.Errorf("access denied")
	}
	return nil
}

func normalizeWhatsAppField(raw string, maxLen int, trimLeadingZeros bool) string {
	out := strings.TrimSpace(raw)
	out = strings.TrimPrefix(out, "+")
	out = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, out)
	if trimLeadingZeros {
		out = strings.TrimLeft(out, "0")
	}
	if len(out) > maxLen {
		out = out[:maxLen]
	}
	return out
}

func (serv *CustomerServImpl) normalizeWhatsAppRequest(request req.CreateWhatsAppCustomerRequest) req.CreateWhatsAppCustomerRequest {
	request.Name = strings.TrimSpace(request.Name)
	request.PhoneCountryCode = normalizeWhatsAppField(request.PhoneCountryCode, 5, true)
	request.PhoneNumber = normalizeWhatsAppField(request.PhoneNumber, 20, true)
	return request
}

func (serv *CustomerServImpl) getClientContext(clientID uuid.UUID) (string, uuid.UUID, error) {
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

func (serv *CustomerServImpl) ensureWhatsAppGuest(schema string, tenantID uuid.UUID, request req.CreateWhatsAppCustomerRequest) (*domains.Guest, error) {
	chatID := request.PhoneCountryCode + request.PhoneNumber
	if chatID == "" {
		return nil, fmt.Errorf("invalid whatsapp identity")
	}

	existing, err := serv.GuestRepo.FindByPlatformChatID(serv.Db, schema, chatID)
	if err == nil && existing != nil {
		updated := false
		if strings.TrimSpace(existing.Name) == "" && request.Name != "" {
			existing.Name = request.Name
			updated = true
		}
		phoneFormatted := "+" + chatID
		if strings.TrimSpace(existing.Phone) == "" {
			existing.Phone = phoneFormatted
			updated = true
		}
		if updated {
			if saveErr := serv.GuestRepo.Update(serv.Db, schema, *existing); saveErr != nil {
				return nil, saveErr
			}
		}
		return existing, nil
	}

	phoneFormatted := "+" + chatID
	guest := domains.Guest{
		TenantID:         &tenantID,
		Identity:         chatID,
		Username:         chatID,
		Phone:            phoneFormatted,
		Name:             request.Name,
		Platform:         "whatsapp",
		PlatformChatID:   chatID,
		Sosmed: domains.JSONB{
			"wa_id": chatID,
			"name":  request.Name,
		},
		IsTakeOver:        false,
		IsRead:            true,
		IsActive:          true,
		ConversationState: domains.JSONB{"state": "registered"},
	}
	if err := serv.GuestRepo.Create(serv.Db, schema, guest); err != nil {
		return nil, err
	}

	created, err := serv.GuestRepo.FindByPlatformChatID(serv.Db, schema, chatID)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (serv *CustomerServImpl) ensureTelegramGuest(schema string, tenantID uuid.UUID, request req.CreateTelegramCustomerRequest) (*domains.Guest, error) {
	username := normalizeTelegramUsername(request.Username)
	if username == "" {
		return nil, fmt.Errorf("invalid telegram username")
	}

	existing, err := serv.GuestRepo.FindByUsername(serv.Db, schema, username)
	if err == nil && existing != nil {
		updated := false
		if strings.TrimSpace(existing.Name) == "" && strings.TrimSpace(request.Name) != "" {
			existing.Name = strings.TrimSpace(request.Name)
			updated = true
		}
		if strings.TrimSpace(existing.Platform) == "" {
			existing.Platform = "telegram"
			updated = true
		}
		if existing.TenantID == nil || *existing.TenantID == uuid.Nil {
			existing.TenantID = &tenantID
			updated = true
		}
		if existing.ConversationState == nil {
			existing.ConversationState = domains.JSONB{"state": "registered_pending_start"}
			updated = true
		}
		if updated {
			if saveErr := serv.GuestRepo.Update(serv.Db, schema, *existing); saveErr != nil {
				return nil, saveErr
			}
		}
		return existing, nil
	}

	guest := domains.Guest{
		TenantID:       &tenantID,
		Identity:       username,
		Username:       username,
		Phone:          "",
		Name:           strings.TrimSpace(request.Name),
		Platform:       "telegram",
		PlatformChatID: "",
		Sosmed: domains.JSONB{
			"username": username,
			"name":     strings.TrimSpace(request.Name),
		},
		IsTakeOver:        false,
		IsRead:            true,
		IsActive:          true,
		ConversationState: domains.JSONB{"state": "registered_pending_start"},
	}
	if err := serv.GuestRepo.Create(serv.Db, schema, guest); err != nil {
		return nil, err
	}

	created, err := serv.GuestRepo.FindByUsername(serv.Db, schema, username)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (serv *CustomerServImpl) findGuestForCustomer(schema string, customer *domains.Customer) (*domains.Guest, error) {
	if customer == nil {
		return nil, fmt.Errorf("customer not found")
	}
	switch strings.ToLower(strings.TrimSpace(customer.AccountType)) {
	case "telegram":
		if customer.Username == nil || normalizeTelegramUsername(*customer.Username) == "" {
			return nil, gorm.ErrRecordNotFound
		}
		return serv.GuestRepo.FindByUsername(serv.Db, schema, normalizeTelegramUsername(*customer.Username))
	case "whatsapp":
		if customer.PhoneCountryCode == nil || customer.PhoneNumber == nil {
			return nil, gorm.ErrRecordNotFound
		}
		cc := normalizeWhatsAppField(*customer.PhoneCountryCode, 5, true)
		num := normalizeWhatsAppField(*customer.PhoneNumber, 20, true)
		if cc == "" || num == "" {
			return nil, gorm.ErrRecordNotFound
		}
		return serv.GuestRepo.FindByPlatformChatID(serv.Db, schema, cc+num)
	default:
		return nil, gorm.ErrRecordNotFound
	}
}

func (serv *CustomerServImpl) guestHasMessages(schema string, guest *domains.Guest) (bool, error) {
	if guest == nil {
		return false, nil
	}
	rows, err := serv.GuestMessageRepo.FindByGuestID(serv.Db, schema, guest.ID, 1)
	if err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}

func (serv *CustomerServImpl) repurposeGuestToTelegram(guest *domains.Guest, tenantID uuid.UUID, request req.CreateTelegramCustomerRequest) {
	username := normalizeTelegramUsername(request.Username)
	guest.TenantID = &tenantID
	guest.Identity = username
	guest.Username = username
	guest.Phone = ""
	guest.Name = strings.TrimSpace(request.Name)
	guest.Platform = "telegram"
	guest.PlatformChatID = ""
	guest.Sosmed = domains.JSONB{
		"username": username,
		"name":     strings.TrimSpace(request.Name),
	}
	guest.ConversationState = domains.JSONB{"state": "registered_pending_start"}
	guest.LastMessageAt = nil
	guest.IsRead = true
	guest.IsActive = true
}

func (serv *CustomerServImpl) repurposeGuestToWhatsApp(guest *domains.Guest, tenantID uuid.UUID, request req.CreateWhatsAppCustomerRequest) {
	chatID := request.PhoneCountryCode + request.PhoneNumber
	guest.TenantID = &tenantID
	guest.Identity = chatID
	guest.Username = chatID
	guest.Phone = "+" + chatID
	guest.Name = strings.TrimSpace(request.Name)
	guest.Platform = "whatsapp"
	guest.PlatformChatID = chatID
	guest.Sosmed = domains.JSONB{
		"wa_id": chatID,
		"name":  strings.TrimSpace(request.Name),
	}
	guest.ConversationState = domains.JSONB{"state": "registered"}
	guest.LastMessageAt = nil
	guest.IsRead = true
	guest.IsActive = true
}

func (serv *CustomerServImpl) attachGuestIDToResponse(schema string, response *res.Response) {
	if response == nil {
		return
	}

	switch strings.ToLower(strings.TrimSpace(response.AccountType)) {
	case "whatsapp":
		if response.PhoneCountryCode == nil || response.PhoneNumber == nil {
			return
		}
		cc := normalizeWhatsAppField(*response.PhoneCountryCode, 5, true)
		num := normalizeWhatsAppField(*response.PhoneNumber, 20, true)
		if cc == "" || num == "" {
			return
		}
		guest, err := serv.GuestRepo.FindByPlatformChatID(serv.Db, schema, cc+num)
		if err == nil && guest != nil {
			guestID := guest.ID.String()
			response.GuestID = &guestID
		}
	case "telegram":
		if response.Username == nil || strings.TrimSpace(*response.Username) == "" {
			return
		}
		username := strings.TrimPrefix(strings.TrimSpace(*response.Username), "@")
		guest, err := serv.GuestRepo.FindByUsername(serv.Db, schema, username)
		if err == nil && guest != nil {
			guestID := guest.ID.String()
			response.GuestID = &guestID
		}
	}
}

func (serv *CustomerServImpl) CreateTelegram(accessToken string, clientID uuid.UUID, request req.CreateTelegramCustomerRequest) (*res.Response, error) {
	if err := serv.checkRole(accessToken); err != nil {
		return nil, err
	}

	if err := serv.checkClientRole(clientID); err != nil {
		return nil, err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return nil, err
	}

	schema, tenantID, err := serv.getClientContext(clientID)
	if err != nil {
		return nil, err
	}

	settings, err := serv.SettingRepo.GetByGroupAndSubGroupName(serv.Db, schema, "integration", "Telegram")
	if err != nil {
		return nil, fmt.Errorf("failed to read telegram integration settings")
	}
	botToken := ""
	for _, s := range settings {
		if s.Name == "telegram-bot-token" {
			botToken = strings.TrimSpace(s.Value)
			break
		}
	}
	if botToken == "" {
		return nil, fmt.Errorf("telegram bot token is not configured")
	}

	username := strings.TrimPrefix(request.Username, "@")
	if !helpers.IsValidTelegramUsername(username) {
		return nil, fmt.Errorf("invalid telegram username: must be 5-32 characters, only letters, numbers, and underscores, and cannot start or end with an underscore")
	}

	// Return existing customer if already registered
	existing, err := serv.CustomerRepo.GetByUsername(serv.Db, schema, username)
	if err == nil && existing != nil {
		response := res.ToResponse(*existing)
		guest, guestErr := serv.ensureTelegramGuest(schema, tenantID, req.CreateTelegramCustomerRequest{
			Name:     existing.Name,
			Username: username,
		})
		if guestErr == nil && guest != nil {
			guestID := guest.ID.String()
			response.GuestID = &guestID
		} else {
			serv.attachGuestIDToResponse(schema, &response)
		}
		return &response, nil
	}

	domain := req.CreateTelegramCustomerToDomain(req.CreateTelegramCustomerRequest{
		Name:     request.Name,
		Username: username,
	})
	customer, err := serv.CustomerRepo.Create(serv.Db, schema, domain)
	if err != nil {
		log.Printf("[CustomerRepo].Create (Telegram) error: %v", err)
		return nil, fmt.Errorf("failed to create customer")
	}

	response := res.ToResponse(*customer)
	guest, guestErr := serv.ensureTelegramGuest(schema, tenantID, req.CreateTelegramCustomerRequest{
		Name:     customer.Name,
		Username: username,
	})
	if guestErr == nil && guest != nil {
		guestID := guest.ID.String()
		response.GuestID = &guestID
	} else {
		serv.attachGuestIDToResponse(schema, &response)
	}
	return &response, nil
}

func (serv *CustomerServImpl) CreateWhatsApp(accessToken string, clientID uuid.UUID, request req.CreateWhatsAppCustomerRequest) (*res.Response, error) {
	if err := serv.checkRole(accessToken); err != nil {
		return nil, err
	}

	if err := serv.checkClientRole(clientID); err != nil {
		return nil, err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return nil, err
	}

	request = serv.normalizeWhatsAppRequest(request)
	if request.Name == "" || request.PhoneCountryCode == "" || request.PhoneNumber == "" {
		return nil, fmt.Errorf("name, phone_country_code, and phone_number are required")
	}

	schema, tenantID, err := serv.getClientContext(clientID)
	if err != nil {
		return nil, err
	}

	// Get WhatsApp credentials from connection table
	waConn, err := serv.WhatsAppConnectionRepo.FindByUserID(serv.Db, clientID)
	if err != nil || waConn == nil {
		return nil, fmt.Errorf("whatsapp integration not configured for this account")
	}

	// Validate WhatsApp phone number exists
	waClient := helpers.NewWhatsAppClient(waConn.PhoneNumberID, waConn.AccessToken)
	exists, err := waClient.CheckPhoneExists(request.PhoneCountryCode, request.PhoneNumber)
	if err != nil {
		if helpers.IsWhatsAppValidationUnsupported(err) {
			log.Printf("[CustomerServ].CreateWhatsApp skip remote validation: %v", err)
			exists = true
		} else {
		log.Printf("[CustomerServ].CreateWhatsApp check phone error: %v", err)
		return nil, fmt.Errorf("failed to validate whatsapp number")
		}
	}
	if !exists {
		return nil, fmt.Errorf("whatsapp number +%s%s is not registered on WhatsApp", request.PhoneCountryCode, request.PhoneNumber)
	}

	// Return existing customer if already registered
	existing, err := serv.CustomerRepo.GetByPhone(serv.Db, schema, request.PhoneCountryCode, request.PhoneNumber)
	if err == nil && existing != nil {
		response := res.ToResponse(*existing)
		guest, guestErr := serv.ensureWhatsAppGuest(schema, tenantID, request)
		if guestErr == nil && guest != nil {
			guestID := guest.ID.String()
			response.GuestID = &guestID
		}
		return &response, nil
	}

	domain := req.CreateWhatsAppCustomerToDomain(request)
	customer, err := serv.CustomerRepo.Create(serv.Db, schema, domain)
	if err != nil {
		log.Printf("[CustomerRepo].Create (WhatsApp) error: %v", err)
		return nil, fmt.Errorf("failed to create customer")
	}

	response := res.ToResponse(*customer)
	guest, err := serv.ensureWhatsAppGuest(schema, tenantID, request)
	if err != nil {
		log.Printf("[CustomerServ].CreateWhatsApp ensure guest error: %v", err)
	} else if guest != nil {
		guestID := guest.ID.String()
		response.GuestID = &guestID
	}
	return &response, nil
}

func (serv *CustomerServImpl) Update(accessToken string, clientID uuid.UUID, customerID int, request req.CreateCustomerRequest) (*res.Response, error) {
	if err := serv.checkRole(accessToken); err != nil {
		return nil, err
	}
	if err := serv.checkClientRole(clientID); err != nil {
		return nil, err
	}

	schema, tenantID, err := serv.getClientContext(clientID)
	if err != nil {
		return nil, err
	}

	customer, err := serv.CustomerRepo.GetByID(serv.Db, schema, customerID)
	if err != nil || customer == nil {
		return nil, fmt.Errorf("customer not found")
	}

	accountType := strings.TrimSpace(request.AccountType)
	switch accountType {
	case "Telegram", "Whatsapp":
	default:
		return nil, fmt.Errorf("unsupported account_type")
	}

	currentGuest, guestErr := serv.findGuestForCustomer(schema, customer)
	if guestErr != nil && guestErr != gorm.ErrRecordNotFound {
		return nil, guestErr
	}
	hasMessages, err := serv.guestHasMessages(schema, currentGuest)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect customer chat history")
	}
	if !strings.EqualFold(customer.AccountType, accountType) && hasMessages {
		return nil, fmt.Errorf("cannot change account type after chat history exists")
	}

	customer.Name = strings.TrimSpace(request.Name)
	switch accountType {
	case "Telegram":
		username := normalizeTelegramUsername(request.Username)
		currentUsername := ""
		if customer.Username != nil {
			currentUsername = normalizeTelegramUsername(*customer.Username)
		}
		if hasMessages && !strings.EqualFold(currentUsername, username) {
			return nil, fmt.Errorf("cannot change telegram username after chat history exists")
		}
		if customer.Name == "" || username == "" {
			return nil, fmt.Errorf("name and username are required")
		}
		if !helpers.IsValidTelegramUsername(username) {
			return nil, fmt.Errorf("invalid telegram username: must be 5-32 characters, only letters, numbers, and underscores, and cannot start or end with an underscore")
		}
		settings, err := serv.SettingRepo.GetByGroupAndSubGroupName(serv.Db, schema, "integration", "Telegram")
		if err != nil {
			return nil, fmt.Errorf("failed to read telegram integration settings")
		}
		botToken := ""
		for _, s := range settings {
			if s.Name == "telegram-bot-token" {
				botToken = strings.TrimSpace(s.Value)
				break
			}
		}
		if botToken == "" {
			return nil, fmt.Errorf("telegram bot token is not configured")
		}
		customer.AccountType = "Telegram"
		customer.Username = &username
		customer.PhoneCountryCode = nil
		customer.PhoneNumber = nil
		if _, err := serv.CustomerRepo.Update(serv.Db, schema, *customer); err != nil {
			return nil, fmt.Errorf("failed to update customer")
		}
		if currentGuest != nil && !hasMessages {
			serv.repurposeGuestToTelegram(currentGuest, tenantID, req.CreateTelegramCustomerRequest{Name: customer.Name, Username: username})
			if err := serv.GuestRepo.Update(serv.Db, schema, *currentGuest); err != nil {
				return nil, err
			}
		} else if currentGuest == nil {
			if _, err := serv.ensureTelegramGuest(schema, tenantID, req.CreateTelegramCustomerRequest{Name: customer.Name, Username: username}); err != nil {
				return nil, err
			}
		}
	case "Whatsapp":
		waReq := serv.normalizeWhatsAppRequest(req.CreateWhatsAppCustomerRequest{
			Name:             request.Name,
			PhoneCountryCode: request.PhoneCountryCode,
			PhoneNumber:      request.PhoneNumber,
		})
		currentCC := ""
		currentNum := ""
		if customer.PhoneCountryCode != nil {
			currentCC = normalizeWhatsAppField(*customer.PhoneCountryCode, 5, true)
		}
		if customer.PhoneNumber != nil {
			currentNum = normalizeWhatsAppField(*customer.PhoneNumber, 20, true)
		}
		if hasMessages && (currentCC != waReq.PhoneCountryCode || currentNum != waReq.PhoneNumber) {
			return nil, fmt.Errorf("cannot change whatsapp phone number after chat history exists")
		}
		if waReq.Name == "" || waReq.PhoneCountryCode == "" || waReq.PhoneNumber == "" {
			return nil, fmt.Errorf("name, phone_country_code, and phone_number are required")
		}
		waConn, err := serv.WhatsAppConnectionRepo.FindByUserID(serv.Db, clientID)
		if err != nil || waConn == nil {
			return nil, fmt.Errorf("whatsapp integration not configured for this account")
		}
		waClient := helpers.NewWhatsAppClient(waConn.PhoneNumberID, waConn.AccessToken)
		exists, err := waClient.CheckPhoneExists(waReq.PhoneCountryCode, waReq.PhoneNumber)
		if err != nil {
			if helpers.IsWhatsAppValidationUnsupported(err) {
				exists = true
			} else {
				return nil, fmt.Errorf("failed to validate whatsapp number")
			}
		}
		if !exists {
			return nil, fmt.Errorf("whatsapp number +%s%s is not registered on WhatsApp", waReq.PhoneCountryCode, waReq.PhoneNumber)
		}
		customer.AccountType = "Whatsapp"
		customer.Username = nil
		customer.PhoneCountryCode = &waReq.PhoneCountryCode
		customer.PhoneNumber = &waReq.PhoneNumber
		if _, err := serv.CustomerRepo.Update(serv.Db, schema, *customer); err != nil {
			return nil, fmt.Errorf("failed to update customer")
		}
		if currentGuest != nil && !hasMessages {
			serv.repurposeGuestToWhatsApp(currentGuest, tenantID, waReq)
			if err := serv.GuestRepo.Update(serv.Db, schema, *currentGuest); err != nil {
				return nil, err
			}
		} else if currentGuest == nil {
			if _, err := serv.ensureWhatsAppGuest(schema, tenantID, waReq); err != nil {
				return nil, err
			}
		}
	}

	updated, err := serv.CustomerRepo.GetByID(serv.Db, schema, customerID)
	if err != nil || updated == nil {
		return nil, fmt.Errorf("failed to reload customer")
	}
	response := res.ToResponse(*updated)
	serv.attachGuestIDToResponse(schema, &response)
	return &response, nil
}

func (serv *CustomerServImpl) GetAll(accessToken string, clientID uuid.UUID, pg domains.Pagination) (*pagination.Response, error) {
	if err := serv.checkRole(accessToken); err != nil {
		return nil, err
	}

	if err := serv.checkClientRole(clientID); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(clientID)
	if err != nil {
		return nil, err
	}

	customers, total, err := serv.CustomerRepo.GetAll(serv.Db, schema, pg)
	if err != nil {
		log.Printf("[CustomerRepo].GetAll error: %v", err)
		return nil, fmt.Errorf("failed to get customers")
	}

	responses := res.ToResponses(customers)
	for i := range responses {
		serv.attachGuestIDToResponse(schema, &responses[i])
	}
	result := pagination.ToResponse(responses, total, pg.Page, pg.Limit)
	return &result, nil
}

func (serv *CustomerServImpl) GetByID(accessToken string, clientID uuid.UUID, id int) (*res.Response, error) {
	if err := serv.checkRole(accessToken); err != nil {
		return nil, err
	}

	if err := serv.checkClientRole(clientID); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(clientID)
	if err != nil {
		return nil, err
	}

	customer, err := serv.CustomerRepo.GetByID(serv.Db, schema, id)
	if err != nil {
		log.Printf("[CustomerRepo].GetByID error: %v", err)
		return nil, fmt.Errorf("customer not found")
	}

	response := res.ToResponse(*customer)
	serv.attachGuestIDToResponse(schema, &response)
	return &response, nil
}
