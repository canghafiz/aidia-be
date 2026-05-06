package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	"crypto/rand"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"math/big"
	"gorm.io/gorm"
)

type WhatsAppOAuthContImpl struct {
	WhatsAppConnectionRepo repositories.WhatsAppConnectionRepo
	SettingRepo            repositories.SettingRepo
	Db                     *gorm.DB
	JwtKey                 string
}

type waRegistrationStatus struct {
	Ready   bool
	Message string
}

func (cont *WhatsAppOAuthContImpl) generateRegistrationPin() string {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "123456"
	}
	return strconv.FormatInt(n.Int64()+100000, 10)
}

func NewWhatsAppOAuthContImpl(
	whatsAppConnectionRepo repositories.WhatsAppConnectionRepo,
	settingRepo repositories.SettingRepo,
	db *gorm.DB,
	jwtKey string,
) *WhatsAppOAuthContImpl {
	return &WhatsAppOAuthContImpl{
		WhatsAppConnectionRepo: whatsAppConnectionRepo,
		SettingRepo:            settingRepo,
		Db:                     db,
		JwtKey:                 jwtKey,
	}
}

type connectWhatsAppRequest struct {
	Code          string `json:"code" binding:"required"`  // from Embedded Signup callback
	WabaID        string `json:"waba_id" binding:"required"` // from WA_EMBEDDED_SIGNUP message event
	PhoneNumberID string `json:"phone_number_id"`           // from message event, optional
}

// Connect godoc
// @Summary      Connect WhatsApp Business Account
// @Description  Connect tenant's WhatsApp Business account via Meta Embedded Signup. Frontend sends code + waba_id from Embedded Signup callback, backend exchanges code for access token.
// @Tags         WhatsApp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string                   true  "Client ID"
// @Param        request    body      connectWhatsAppRequest   true  "code and waba_id from Embedded Signup"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/whatsapp/connect [post]
func (cont *WhatsAppOAuthContImpl) Connect(ctx *gin.Context) {
	userIDStr, tenantSchema := cont.getUserContext(ctx)
	if userIDStr == "" || tenantSchema == "" {
		ctx.JSON(401, gin.H{"success": false, "message": "Unauthorized"})
		return
	}

	var req connectWhatsAppRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "message": "code and waba_id are required"})
		return
	}

	// Exchange Embedded Signup code for access token
	// No redirect_uri needed for Embedded Signup (JS SDK flow)
	log.Printf("[WA OAuth] exchanging Embedded Signup code for user=%s waba=%s", userIDStr, req.WabaID)
	accessToken, err := helpers.ExchangeCodeForToken(req.Code)
	if err != nil {
		log.Printf("[WA OAuth] failed to exchange code: %v", err)
		ctx.JSON(400, gin.H{"success": false, "message": "Failed to verify WhatsApp account: " + err.Error()})
		return
	}

	// Extend token to last ~60 days
	extendedToken, err := helpers.ExtendToken(accessToken)
	if err != nil {
		log.Printf("[WA OAuth] failed to extend token, using short-lived token: %v", err)
		extendedToken = accessToken
	}

	// 3. Get phone number ID
	phoneNumberID := req.PhoneNumberID
	phoneNumber := ""
	displayName := ""

	if phoneNumberID == "" {
		// Fetch from WABA if not sent by frontend
		phones, err := helpers.GetWABAPhoneNumbers(req.WabaID, extendedToken)
		if err != nil || len(phones) == 0 {
			log.Printf("[WA OAuth] failed to fetch phone numbers: %v", err)
			ctx.JSON(400, gin.H{"success": false, "message": "Failed to retrieve phone number from WhatsApp Business account"})
			return
		}
		phoneNumberID = phones[0].ID
		phoneNumber = phones[0].DisplayPhoneNumber
		displayName = phones[0].VerifiedName
	} else {
		// Look up number info from WABA list
		phones, err := helpers.GetWABAPhoneNumbers(req.WabaID, extendedToken)
		if err == nil {
			for _, p := range phones {
				if p.ID == phoneNumberID {
					phoneNumber = p.DisplayPhoneNumber
					displayName = p.VerifiedName
					break
				}
			}
		}
	}

	// 4. Subscribe our app to this WABA's webhook events
	if err := helpers.SubscribeAppToWABA(req.WabaID, extendedToken); err != nil {
		log.Printf("[WA OAuth] ⚠️ failed to subscribe webhook to WABA=%s: %v", req.WabaID, err)
	} else {
		log.Printf("[WA OAuth] ✅ webhook subscribed to WABA=%s", req.WabaID)
	}

	registrationStatus := waRegistrationStatus{Ready: true}
	if pin := cont.ensureTenantRegistrationPin(tenantSchema); pin != "" && phoneNumberID != "" {
		waClient := helpers.NewWhatsAppClient(phoneNumberID, extendedToken)
		if err := waClient.RegisterPhoneNumber(pin); err != nil {
			log.Printf("[WA OAuth] failed to register phone_number_id=%s: %v", phoneNumberID, err)
			registrationStatus.Ready = false
			if helpers.IsWhatsAppRegistrationBlocked(err) {
				registrationStatus.Message = "Connected, but this number is still attached to another WhatsApp account and cannot send messages yet."
			} else {
				registrationStatus.Message = "Connected, but the WhatsApp business number is not ready to send messages yet."
			}
		} else {
			log.Printf("[WA OAuth] WhatsApp number successfully registered phone_number_id=%s", phoneNumberID)
			registrationStatus.Message = ""
		}
	}

	// 5. Save to public.whatsapp_connections
	userID, _ := uuid.Parse(userIDStr)
	now := time.Now()
	conn := domains.WhatsAppConnection{
		UserID:        userID,
		TenantSchema:  tenantSchema,
		PhoneNumberID: phoneNumberID,
		WabaID:        req.WabaID,
		AccessToken:   extendedToken,
		PhoneNumber:   phoneNumber,
		DisplayName:   displayName,
		ConnectedAt:   now,
	}

	if err := cont.WhatsAppConnectionRepo.Upsert(cont.Db, conn); err != nil {
		log.Printf("[WA OAuth] failed to save connection: %v", err)
		ctx.JSON(500, gin.H{"success": false, "message": "Failed to save connection"})
		return
	}

	// 6. Sync to tenant setting table so getWhatsAppClient() keeps working
	cont.syncToTenantSettings(tenantSchema, phoneNumberID, extendedToken)
	cont.updateRegistrationStatus(tenantSchema, registrationStatus)

	log.Printf("[WA OAuth] ✅ successfully connected WhatsApp for schema=%s phone_number_id=%s", tenantSchema, phoneNumberID)

	ctx.JSON(200, gin.H{
		"success": true,
		"code":    200,
		"data": gin.H{
			"phone_number_id": phoneNumberID,
			"phone_number":    phoneNumber,
			"display_name":    displayName,
			"waba_id":         req.WabaID,
			"connected_at":    now,
			"ready_to_send":   registrationStatus.Ready,
			"status_message":  registrationStatus.Message,
		},
	})
}

// Status godoc
// @Summary      Get WhatsApp Connection Status
// @Description  Check whether the tenant's WhatsApp Business account is connected. Returns phone number and display name if connected.
// @Tags         WhatsApp
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string  true  "Client ID"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/whatsapp/status [get]
func (cont *WhatsAppOAuthContImpl) Status(ctx *gin.Context) {
	userIDStr, tenantSchema := cont.getUserContext(ctx)
	if userIDStr == "" {
		ctx.JSON(401, gin.H{"success": false, "message": "Unauthorized"})
		return
	}

	userID, _ := uuid.Parse(userIDStr)
	conn, err := cont.WhatsAppConnectionRepo.FindByUserID(cont.Db, userID)
	if err != nil {
		// Not connected yet
		ctx.JSON(200, gin.H{
			"success": true,
			"code":    200,
			"data": gin.H{
				"connected": false,
			},
		})
		return
	}

	registrationStatus := waRegistrationStatus{Ready: true}
	if tenantSchema != "" {
		registrationStatus = cont.readRegistrationStatus(tenantSchema)
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"code":    200,
		"data": gin.H{
			"connected":       true,
			"phone_number_id": conn.PhoneNumberID,
			"phone_number":    conn.PhoneNumber,
			"display_name":    conn.DisplayName,
			"waba_id":         conn.WabaID,
			"connected_at":    conn.ConnectedAt,
			"ready_to_send":   registrationStatus.Ready,
			"status_message":  registrationStatus.Message,
		},
	})
}

// Disconnect godoc
// @Summary      Disconnect WhatsApp Business Account
// @Description  Disconnect the tenant's WhatsApp Business account and clear stored credentials.
// @Tags         WhatsApp
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string  true  "Client ID"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/whatsapp/disconnect [delete]
func (cont *WhatsAppOAuthContImpl) Disconnect(ctx *gin.Context) {
	userIDStr, tenantSchema := cont.getUserContext(ctx)
	if userIDStr == "" || tenantSchema == "" {
		ctx.JSON(401, gin.H{"success": false, "message": "Unauthorized"})
		return
	}

	userID, _ := uuid.Parse(userIDStr)

	if err := cont.WhatsAppConnectionRepo.DeleteByUserID(cont.Db, userID); err != nil {
		log.Printf("[WA OAuth] failed to disconnect for user=%s: %v", userIDStr, err)
		ctx.JSON(500, gin.H{"success": false, "message": "Failed to disconnect"})
		return
	}

	// Clear credentials from tenant setting table
	cont.clearTenantSettings(tenantSchema)

	log.Printf("[WA OAuth] WhatsApp connection disconnected for schema=%s", tenantSchema)

	ctx.JSON(200, gin.H{
		"success": true,
		"code":    200,
		"data":    gin.H{"message": "WhatsApp connection successfully disconnected"},
	})
}

// GetAuthURL godoc
// @Summary      Get WhatsApp OAuth URL
// @Description  Return public Meta config needed by frontend to init FB SDK for Embedded Signup.
// @Tags         WhatsApp
// @Produce      json
// @Success      200  {object}  helpers.ApiResponse
// @Router       /whatsapp/config [get]
func (cont *WhatsAppOAuthContImpl) GetConfig(ctx *gin.Context) {
	appID := os.Getenv("META_APP_ID")
	configID := os.Getenv("META_EMBEDDED_SIGNUP_CONFIG_ID")
	if appID == "" {
		ctx.JSON(500, gin.H{"success": false, "message": "META_APP_ID is not configured"})
		return
	}
	ctx.JSON(200, gin.H{
		"success": true,
		"code":    200,
		"data": gin.H{
			"app_id":    appID,
			"config_id": configID,
		},
	})
}

// GetAuthURL — deprecated, replaced by Embedded Signup
func (cont *WhatsAppOAuthContImpl) GetAuthURL(ctx *gin.Context) {
	userIDStr, tenantSchema := cont.getUserContext(ctx)
	if userIDStr == "" || tenantSchema == "" {
		ctx.JSON(401, gin.H{"success": false, "message": "Unauthorized"})
		return
	}

	appID := os.Getenv("META_APP_ID")
	callbackURL := os.Getenv("META_OAUTH_CALLBACK_URL")

	if appID == "" || callbackURL == "" {
		ctx.JSON(500, gin.H{"success": false, "message": "META_APP_ID or META_OAUTH_CALLBACK_URL is not configured"})
		return
	}

	// State token contains user_id + tenant_schema, valid for 10 minutes (anti-CSRF)
	stateData := map[string]interface{}{
		"user_id":       userIDStr,
		"tenant_schema": tenantSchema,
	}
	stateToken, err := helpers.GenerateJWT(cont.JwtKey, 10*time.Minute, stateData)
	if err != nil {
		ctx.JSON(500, gin.H{"success": false, "message": "Failed to generate state token"})
		return
	}

	loginPlatform := ctx.DefaultQuery("platform", "facebook") // "facebook" | "instagram"

	params := url.Values{}
	params.Set("client_id", appID)
	params.Set("redirect_uri", callbackURL)
	params.Set("state", stateToken)
	params.Set("response_type", "code")
	params.Set("auth_type", "rerequest")

	var authURL string
	if loginPlatform == "instagram" {
		// Instagram Business Login — uses instagram.com OAuth entry point
		// Accesses the WABA linked to the Instagram Business Account
		params.Set("scope", "whatsapp_business_management,whatsapp_business_messaging,business_management,instagram_basic,instagram_manage_insights")
		authURL = "https://www.instagram.com/oauth/authorize?" + params.Encode()
	} else {
		params.Set("scope", "whatsapp_business_management,whatsapp_business_messaging,business_management")
		authURL = "https://www.facebook.com/dialog/oauth?" + params.Encode()
	}

	ctx.JSON(200, gin.H{
		"success": true,
		"code":    200,
		"data":    gin.H{"auth_url": authURL},
	})
}

// ConnectWithSession godoc
// @Summary      Connect WhatsApp using OAuth session token
// @Description  Used after OAuth callback when WABA is not found automatically. Client submits session token (from ?session= URL param) + WABA ID manually.
// @Tags         WhatsApp
// @Accept       json
// @Produce      json
// @Param        request  body  connectWithSessionRequest  true  "Session token and WABA ID"
// @Success      200  {object}  helpers.ApiResponse
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      500  {object}  helpers.ApiResponse
// @Router       /whatsapp/connect-with-session [post]
func (cont *WhatsAppOAuthContImpl) ConnectWithSession(ctx *gin.Context) {
	var req connectWithSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || req.SessionToken == "" || req.WabaID == "" {
		ctx.JSON(400, gin.H{"success": false, "message": "session_token and waba_id are required"})
		return
	}

	// Decode session token
	sessionData, err := helpers.DecodeJWT(req.SessionToken, cont.JwtKey)
	if err != nil {
		ctx.JSON(401, gin.H{"success": false, "message": "Session token is invalid or expired"})
		return
	}
	accessToken, _ := sessionData["access_token"].(string)
	userIDStr, _   := sessionData["user_id"].(string)
	tenantSchema, _ := sessionData["tenant_schema"].(string)

	if accessToken == "" || userIDStr == "" || tenantSchema == "" {
		ctx.JSON(400, gin.H{"success": false, "message": "Session data is incomplete"})
		return
	}

	// Get phone number from the WABA entered by the user
	phones, err := helpers.GetWABAPhoneNumbers(req.WabaID, accessToken)
	if err != nil || len(phones) == 0 {
		ctx.JSON(400, gin.H{"success": false, "message": "Failed to retrieve phone number from WABA: " + req.WabaID})
		return
	}

	phoneNumberID := phones[0].ID
	phoneNumber   := phones[0].DisplayPhoneNumber
	displayName   := phones[0].VerifiedName

	if req.PhoneNumberID != "" {
		for _, p := range phones {
			if p.ID == req.PhoneNumberID {
				phoneNumberID = p.ID
				phoneNumber   = p.DisplayPhoneNumber
				displayName   = p.VerifiedName
				break
			}
		}
	}

	// Subscribe webhook
	if err := helpers.SubscribeAppToWABA(req.WabaID, accessToken); err != nil {
		log.Printf("[WA Session Connect] failed to subscribe webhook WABA=%s: %v", req.WabaID, err)
	}

	// Save to DB
	userID, _ := uuid.Parse(userIDStr)
	now := time.Now()
	conn := domains.WhatsAppConnection{
		UserID:        userID,
		TenantSchema:  tenantSchema,
		PhoneNumberID: phoneNumberID,
		WabaID:        req.WabaID,
		AccessToken:   accessToken,
		PhoneNumber:   phoneNumber,
		DisplayName:   displayName,
		ConnectedAt:   now,
	}
	if err := cont.WhatsAppConnectionRepo.Upsert(cont.Db, conn); err != nil {
		ctx.JSON(500, gin.H{"success": false, "message": "Failed to save connection"})
		return
	}
	cont.syncToTenantSettings(tenantSchema, phoneNumberID, accessToken)
	if pin := cont.ensureTenantRegistrationPin(tenantSchema); pin != "" && phoneNumberID != "" {
		waClient := helpers.NewWhatsAppClient(phoneNumberID, accessToken)
		registrationStatus := waRegistrationStatus{Ready: true}
		if err := waClient.RegisterPhoneNumber(pin); err != nil {
			log.Printf("[WA Session Connect] failed to register phone_number_id=%s: %v", phoneNumberID, err)
			registrationStatus.Ready = false
			if helpers.IsWhatsAppRegistrationBlocked(err) {
				registrationStatus.Message = "Connected, but this number is still attached to another WhatsApp account and cannot send messages yet."
			} else {
				registrationStatus.Message = "Connected, but the WhatsApp business number is not ready to send messages yet."
			}
		} else {
			log.Printf("[WA Session Connect] WhatsApp number successfully registered phone_number_id=%s", phoneNumberID)
		}
		cont.updateRegistrationStatus(tenantSchema, registrationStatus)
	}

	log.Printf("[WA Session Connect] ✅ connected schema=%s phone=%s waba=%s", tenantSchema, phoneNumber, req.WabaID)

	ctx.JSON(200, gin.H{
		"success": true,
		"code":    200,
		"data": gin.H{
			"phone_number_id": phoneNumberID,
			"phone_number":    phoneNumber,
			"display_name":    displayName,
			"waba_id":         req.WabaID,
			"connected_at":    now,
		},
	})
}

type connectWithSessionRequest struct {
	SessionToken  string `json:"session_token" binding:"required"`
	WabaID        string `json:"waba_id" binding:"required"`
	PhoneNumberID string `json:"phone_number_id"`
}

// OAuthRedirect redirects the browser directly to Meta OAuth login page.
// Accepts JWT via ?token= query param so it can be called from a plain <a> link or window.location.
// GET /client/{client_id}/whatsapp/oauth-redirect?token=JWT (public-ish, token validated inside)
func (cont *WhatsAppOAuthContImpl) OAuthRedirect(ctx *gin.Context) {
	tokenStr := ctx.Query("token")
	if tokenStr == "" {
		ctx.JSON(400, gin.H{"success": false, "message": "token query param required"})
		return
	}

	claims, err := helpers.DecodeJWT(tokenStr, cont.JwtKey)
	if err != nil {
		ctx.JSON(401, gin.H{"success": false, "message": "Token is invalid or expired"})
		return
	}

	userIDStr, _ := claims["user_id"].(string)
	tenantSchema, _ := claims["tenant_schema"].(string)
	if userIDStr == "" || tenantSchema == "" {
		ctx.JSON(401, gin.H{"success": false, "message": "Token does not contain user context"})
		return
	}

	appID := os.Getenv("META_APP_ID")
	callbackURL := os.Getenv("META_OAUTH_CALLBACK_URL")
	if appID == "" || callbackURL == "" {
		ctx.JSON(500, gin.H{"success": false, "message": "META_APP_ID or META_OAUTH_CALLBACK_URL is not configured"})
		return
	}

	stateData := map[string]interface{}{
		"user_id":       userIDStr,
		"tenant_schema": tenantSchema,
	}
	stateToken, err := helpers.GenerateJWT(cont.JwtKey, 10*time.Minute, stateData)
	if err != nil {
		ctx.JSON(500, gin.H{"success": false, "message": "Failed to generate state token"})
		return
	}

	loginPlatform := ctx.DefaultQuery("platform", "facebook") // "facebook" | "instagram"

	params := url.Values{}
	params.Set("client_id", appID)
	params.Set("redirect_uri", callbackURL)
	params.Set("state", stateToken)
	params.Set("response_type", "code")
	params.Set("auth_type", "rerequest")

	var oauthURL string
	if loginPlatform == "instagram" {
		params.Set("scope", "whatsapp_business_management,whatsapp_business_messaging,business_management,instagram_basic,instagram_manage_insights")
		oauthURL = "https://www.instagram.com/oauth/authorize?" + params.Encode()
	} else {
		params.Set("scope", "whatsapp_business_management,whatsapp_business_messaging,business_management")
		oauthURL = "https://www.facebook.com/dialog/oauth?" + params.Encode()
	}

	ctx.Redirect(302, oauthURL)
}

// OAuthCallback receives the redirect from Meta after the user logs in and grants permission.
// Backend automatically: exchange code → extend token → find WABA → get number → save → redirect to FE.
// GET /api/v1/webhook/whatsapp/oauth-callback (public, no auth)
func (cont *WhatsAppOAuthContImpl) OAuthCallback(ctx *gin.Context) {
	frontendURL := os.Getenv("META_FRONTEND_REDIRECT_URL")
	if frontendURL == "" {
		frontendURL = "/"
	}

	redirectError := func(msg string) {
		ctx.Redirect(302, frontendURL+"?wa_status=error&message="+url.QueryEscape(msg))
	}

	// 1. Get code + state from query params
	code := ctx.Query("code")
	stateToken := ctx.Query("state")
	if code == "" || stateToken == "" {
		redirectError("Incomplete parameters from Meta")
		return
	}

	// 2. Validate state token → ensure request is legitimate and extract user context
	stateData, err := helpers.DecodeJWT(stateToken, cont.JwtKey)
	if err != nil {
		redirectError("State token is invalid or expired")
		return
	}
	userIDStr, _ := stateData["user_id"].(string)
	tenantSchema, _ := stateData["tenant_schema"].(string)
	if userIDStr == "" || tenantSchema == "" {
		redirectError("State token does not contain valid data")
		return
	}

	callbackURL := os.Getenv("META_OAUTH_CALLBACK_URL")

	// 3. Exchange code for access token
	accessToken, err := helpers.ExchangeCodeForTokenWithURI(code, callbackURL)
	if err != nil {
		log.Printf("[WA Callback] failed to exchange code for user=%s: %v", userIDStr, err)
		redirectError("Failed to connect WhatsApp account")
		return
	}

	// 4. Extend token to last ~60 days
	extendedToken, err := helpers.ExtendToken(accessToken)
	if err != nil {
		log.Printf("[WA Callback] failed to extend token, using short-lived token: %v", err)
		extendedToken = accessToken
	}

	// 5. Get user's WABA list → automatically pick the first one
	wabas, err := helpers.GetUserWABAs(extendedToken)
	if err != nil || len(wabas) == 0 {
		log.Printf("[WA Callback] WABA not found automatically for user=%s, redirecting to manual input: %v", userIDStr, err)

		// Wrap access_token + user context in a short-lived JWT (15 minutes)
		// so the HTML can POST to /connect without exposing the raw token in the URL
		sessionData := map[string]interface{}{
			"access_token":  extendedToken,
			"user_id":       userIDStr,
			"tenant_schema": tenantSchema,
		}
		sessionToken, err := helpers.GenerateJWT(cont.JwtKey, 15*time.Minute, sessionData)
		if err != nil {
			redirectError("Failed to generate session token")
			return
		}
		ctx.Redirect(302, frontendURL+
			"?wa_status=need_waba"+
			"&session="+url.QueryEscape(sessionToken))
		return
	}
	wabaID := wabas[0].ID

	// 6. Get phone number from WABA
	phones, err := helpers.GetWABAPhoneNumbers(wabaID, extendedToken)
	if err != nil || len(phones) == 0 {
		log.Printf("[WA Callback] no phone number found in WABA=%s: %v", wabaID, err)
		redirectError("No phone number found in WhatsApp Business account")
		return
	}
	phoneNumberID := phones[0].ID
	phoneNumber := phones[0].DisplayPhoneNumber
	displayName := phones[0].VerifiedName

	// 7. Subscribe app to WABA webhook events (non-fatal if it fails)
	if err := helpers.SubscribeAppToWABA(wabaID, extendedToken); err != nil {
		log.Printf("[WA Callback] warning: failed to subscribe webhook WABA=%s: %v", wabaID, err)
	}

	// 8. Save connection to DB
	userID, _ := uuid.Parse(userIDStr)
	now := time.Now()
	conn := domains.WhatsAppConnection{
		UserID:        userID,
		TenantSchema:  tenantSchema,
		PhoneNumberID: phoneNumberID,
		WabaID:        wabaID,
		AccessToken:   extendedToken,
		PhoneNumber:   phoneNumber,
		DisplayName:   displayName,
		ConnectedAt:   now,
	}
	if err := cont.WhatsAppConnectionRepo.Upsert(cont.Db, conn); err != nil {
		log.Printf("[WA Callback] failed to save connection schema=%s: %v", tenantSchema, err)
		redirectError("Failed to save connection")
		return
	}

	// 9. Sync to tenant settings so getWhatsAppClient() can use it immediately
	cont.syncToTenantSettings(tenantSchema, phoneNumberID, extendedToken)
	if pin := cont.ensureTenantRegistrationPin(tenantSchema); pin != "" && phoneNumberID != "" {
		waClient := helpers.NewWhatsAppClient(phoneNumberID, extendedToken)
		registrationStatus := waRegistrationStatus{Ready: true}
		if err := waClient.RegisterPhoneNumber(pin); err != nil {
			log.Printf("[WA Callback] failed to register phone_number_id=%s: %v", phoneNumberID, err)
			registrationStatus.Ready = false
			if helpers.IsWhatsAppRegistrationBlocked(err) {
				registrationStatus.Message = "Connected, but this number is still attached to another WhatsApp account and cannot send messages yet."
			} else {
				registrationStatus.Message = "Connected, but the WhatsApp business number is not ready to send messages yet."
			}
		} else {
			log.Printf("[WA Callback] WhatsApp number successfully registered phone_number_id=%s", phoneNumberID)
		}
		cont.updateRegistrationStatus(tenantSchema, registrationStatus)
	}

	log.Printf("[WA Callback] ✅ connected schema=%s phone=%s waba=%s", tenantSchema, phoneNumber, wabaID)

	// 10. Redirect to FE with success info
	ctx.Redirect(302, frontendURL+
		"?wa_status=connected"+
		"&phone="+url.QueryEscape(phoneNumber)+
		"&display_name="+url.QueryEscape(displayName))
}

// getUserContext extracts user_id and tenant_schema from the JWT
func (cont *WhatsAppOAuthContImpl) getUserContext(ctx *gin.Context) (userID, tenantSchema string) {
	userID, _ = ctx.MustGet("user_id").(string)

	header := ctx.GetHeader("Authorization")
	tokenStr := strings.TrimPrefix(header, "Bearer ")
	claims, err := helpers.DecodeJWT(tokenStr, cont.JwtKey)
	if err != nil {
		return userID, ""
	}

	tenantSchema, _ = claims["tenant_schema"].(string)
	return userID, tenantSchema
}

// syncToTenantSettings syncs credentials to the tenant setting table
// so that existing getWhatsAppClient() logic continues to work
func (cont *WhatsAppOAuthContImpl) syncToTenantSettings(schema, phoneNumberID, accessToken string) {
	type settingKV struct {
		name  string
		value string
	}

	settings := []settingKV{
		{name: "whatsapp-phone-number-id", value: phoneNumberID},
		{name: "whatsapp-access-token", value: accessToken},
	}

	for _, s := range settings {
		err := cont.Db.Exec(
			`INSERT INTO `+schema+`.setting (id, group_name, sub_group_name, name, value, created_at, updated_at)
			VALUES (gen_random_uuid(), 'integration', 'WhatsApp', ?, ?, NOW(), NOW())
			ON CONFLICT (sub_group_name, name) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			s.name, s.value,
		).Error
		if err != nil {
			log.Printf("[WA OAuth] failed to sync setting %s to schema=%s: %v", s.name, schema, err)
		}
	}
}

func (cont *WhatsAppOAuthContImpl) ensureTenantRegistrationPin(schema string) string {
	settings, err := cont.SettingRepo.GetByGroupAndSubGroupName(cont.Db, schema, "integration", "WhatsApp")
	if err == nil {
		for _, s := range settings {
			if s.Name == "whatsapp-registration-pin" && strings.TrimSpace(s.Value) != "" {
				return strings.TrimSpace(s.Value)
			}
		}
	}

	pin := cont.generateRegistrationPin()
	err = cont.Db.Exec(
		`INSERT INTO `+schema+`.setting (id, group_name, sub_group_name, name, value, created_at, updated_at)
		VALUES (gen_random_uuid(), 'integration', 'WhatsApp', 'whatsapp-registration-pin', ?, NOW(), NOW())
		ON CONFLICT (sub_group_name, name) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
		pin,
	).Error
	if err != nil {
		log.Printf("[WA OAuth] failed to save registration pin schema=%s: %v", schema, err)
		return ""
	}

	return pin
}

func (cont *WhatsAppOAuthContImpl) updateRegistrationStatus(schema string, status waRegistrationStatus) {
	values := []struct {
		name  string
		value string
	}{
		{name: "whatsapp-ready-to-send", value: strconv.FormatBool(status.Ready)},
		{name: "whatsapp-status-message", value: strings.TrimSpace(status.Message)},
	}

	for _, s := range values {
		err := cont.Db.Exec(
			`INSERT INTO `+schema+`.setting (id, group_name, sub_group_name, name, value, created_at, updated_at)
			VALUES (gen_random_uuid(), 'integration', 'WhatsApp', ?, ?, NOW(), NOW())
			ON CONFLICT (sub_group_name, name) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			s.name, s.value,
		).Error
		if err != nil {
			log.Printf("[WA OAuth] failed to sync status %s to schema=%s: %v", s.name, schema, err)
		}
	}
}

func (cont *WhatsAppOAuthContImpl) readRegistrationStatus(schema string) waRegistrationStatus {
	status := waRegistrationStatus{Ready: true}
	settings, err := cont.SettingRepo.GetByGroupAndSubGroupName(cont.Db, schema, "integration", "WhatsApp")
	if err != nil {
		return status
	}
	for _, s := range settings {
		switch s.Name {
		case "whatsapp-ready-to-send":
			status.Ready = strings.EqualFold(strings.TrimSpace(s.Value), "true")
		case "whatsapp-status-message":
			status.Message = strings.TrimSpace(s.Value)
		}
	}
	return status
}

// clearTenantSettings clears WhatsApp credentials from the tenant setting table
func (cont *WhatsAppOAuthContImpl) clearTenantSettings(schema string) {
	err := cont.Db.Exec(
		`UPDATE `+schema+`.setting SET value = '', updated_at = NOW()
		WHERE sub_group_name = 'WhatsApp' AND name IN ('whatsapp-phone-number-id', 'whatsapp-access-token', 'whatsapp-registration-pin', 'whatsapp-ready-to-send', 'whatsapp-status-message')`,
	).Error
	if err != nil {
		log.Printf("[WA OAuth] failed to clear settings for schema=%s: %v", schema, err)
	}
}
