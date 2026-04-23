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
	Code          string `json:"code" binding:"required"`  // dari Embedded Signup callback
	WabaID        string `json:"waba_id" binding:"required"` // dari message event WA_EMBEDDED_SIGNUP
	PhoneNumberID string `json:"phone_number_id"`           // dari message event, opsional
}

// Connect godoc
// @Summary      Connect WhatsApp Business Account
// @Description  Connect tenant's WhatsApp Business account via Meta Embedded Signup. Frontend sends code + waba_id dari Embedded Signup callback, backend tukar code dengan access token.
// @Tags         WhatsApp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string                   true  "Client ID"
// @Param        request    body      connectWhatsAppRequest   true  "code dan waba_id dari Embedded Signup"
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
		ctx.JSON(400, gin.H{"success": false, "message": "code dan waba_id wajib diisi"})
		return
	}

	// Tukar code dari Embedded Signup dengan access token
	// Tidak perlu redirect_uri untuk Embedded Signup (JS SDK flow)
	log.Printf("[WA OAuth] menukar Embedded Signup code untuk user=%s waba=%s", userIDStr, req.WabaID)
	accessToken, err := helpers.ExchangeCodeForToken(req.Code)
	if err != nil {
		log.Printf("[WA OAuth] gagal exchange code: %v", err)
		ctx.JSON(400, gin.H{"success": false, "message": "Gagal verifikasi akun WhatsApp: " + err.Error()})
		return
	}

	// Extend token agar tahan ~60 hari
	extendedToken, err := helpers.ExtendToken(accessToken)
	if err != nil {
		log.Printf("[WA OAuth] gagal extend token, pakai token pendek: %v", err)
		extendedToken = accessToken
	}

	// 3. Ambil phone number ID
	phoneNumberID := req.PhoneNumberID
	phoneNumber := ""
	displayName := ""

	if phoneNumberID == "" {
		// Fetch dari WABA jika tidak dikirim dari frontend
		phones, err := helpers.GetWABAPhoneNumbers(req.WabaID, extendedToken)
		if err != nil || len(phones) == 0 {
			log.Printf("[WA OAuth] gagal ambil phone numbers: %v", err)
			ctx.JSON(400, gin.H{"success": false, "message": "Gagal mendapatkan nomor telepon dari akun WhatsApp Business"})
			return
		}
		phoneNumberID = phones[0].ID
		phoneNumber = phones[0].DisplayPhoneNumber
		displayName = phones[0].VerifiedName
	} else {
		// Cari info nomor dari daftar WABA
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

	// 4. Subscribe app kita ke webhook events WABA ini
	if err := helpers.SubscribeAppToWABA(req.WabaID, extendedToken); err != nil {
		log.Printf("[WA OAuth] ⚠️ gagal subscribe webhook ke WABA=%s: %v", req.WabaID, err)
	} else {
		log.Printf("[WA OAuth] ✅ webhook subscribed ke WABA=%s", req.WabaID)
	}

	registrationStatus := waRegistrationStatus{Ready: true}
	if pin := cont.ensureTenantRegistrationPin(tenantSchema); pin != "" && phoneNumberID != "" {
		waClient := helpers.NewWhatsAppClient(phoneNumberID, extendedToken)
		if err := waClient.RegisterPhoneNumber(pin); err != nil {
			log.Printf("[WA OAuth] gagal register phone_number_id=%s: %v", phoneNumberID, err)
			registrationStatus.Ready = false
			if helpers.IsWhatsAppRegistrationBlocked(err) {
				registrationStatus.Message = "Connected, but this number is still attached to another WhatsApp account and cannot send messages yet."
			} else {
				registrationStatus.Message = "Connected, but the WhatsApp business number is not ready to send messages yet."
			}
		} else {
			log.Printf("[WA OAuth] nomor WhatsApp berhasil diregister phone_number_id=%s", phoneNumberID)
			registrationStatus.Message = ""
		}
	}

	// 5. Simpan ke public.whatsapp_connections
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
		log.Printf("[WA OAuth] gagal simpan koneksi: %v", err)
		ctx.JSON(500, gin.H{"success": false, "message": "Gagal menyimpan koneksi"})
		return
	}

	// 6. Sinkronisasi ke tenant setting table agar getWhatsAppClient() tetap berfungsi
	cont.syncToTenantSettings(tenantSchema, phoneNumberID, extendedToken)
	cont.updateRegistrationStatus(tenantSchema, registrationStatus)

	log.Printf("[WA OAuth] ✅ berhasil hubungkan WhatsApp untuk schema=%s phone_number_id=%s", tenantSchema, phoneNumberID)

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
		// Belum terhubung
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
		log.Printf("[WA OAuth] gagal disconnect untuk user=%s: %v", userIDStr, err)
		ctx.JSON(500, gin.H{"success": false, "message": "Gagal memutuskan koneksi"})
		return
	}

	// Hapus credentials dari tenant setting table
	cont.clearTenantSettings(tenantSchema)

	log.Printf("[WA OAuth] koneksi WhatsApp diputus untuk schema=%s", tenantSchema)

	ctx.JSON(200, gin.H{
		"success": true,
		"code":    200,
		"data":    gin.H{"message": "Koneksi WhatsApp berhasil diputus"},
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
		ctx.JSON(500, gin.H{"success": false, "message": "META_APP_ID belum dikonfigurasi"})
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

// GetAuthURL — deprecated, diganti Embedded Signup
func (cont *WhatsAppOAuthContImpl) GetAuthURL(ctx *gin.Context) {
	userIDStr, tenantSchema := cont.getUserContext(ctx)
	if userIDStr == "" || tenantSchema == "" {
		ctx.JSON(401, gin.H{"success": false, "message": "Unauthorized"})
		return
	}

	appID := os.Getenv("META_APP_ID")
	callbackURL := os.Getenv("META_OAUTH_CALLBACK_URL")

	if appID == "" || callbackURL == "" {
		ctx.JSON(500, gin.H{"success": false, "message": "META_APP_ID atau META_OAUTH_CALLBACK_URL belum dikonfigurasi"})
		return
	}

	// State token berisi user_id + tenant_schema, berlaku 10 menit (anti-CSRF)
	stateData := map[string]interface{}{
		"user_id":       userIDStr,
		"tenant_schema": tenantSchema,
	}
	stateToken, err := helpers.GenerateJWT(cont.JwtKey, 10*time.Minute, stateData)
	if err != nil {
		ctx.JSON(500, gin.H{"success": false, "message": "Gagal generate state token"})
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
		// Mengakses WABA yang terhubung ke Instagram Business Account
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
		ctx.JSON(400, gin.H{"success": false, "message": "session_token dan waba_id wajib diisi"})
		return
	}

	// Decode session token
	sessionData, err := helpers.DecodeJWT(req.SessionToken, cont.JwtKey)
	if err != nil {
		ctx.JSON(401, gin.H{"success": false, "message": "Session token tidak valid atau sudah kadaluarsa"})
		return
	}
	accessToken, _ := sessionData["access_token"].(string)
	userIDStr, _   := sessionData["user_id"].(string)
	tenantSchema, _ := sessionData["tenant_schema"].(string)

	if accessToken == "" || userIDStr == "" || tenantSchema == "" {
		ctx.JSON(400, gin.H{"success": false, "message": "Session tidak lengkap"})
		return
	}

	// Ambil nomor telepon dari WABA yang diinput user
	phones, err := helpers.GetWABAPhoneNumbers(req.WabaID, accessToken)
	if err != nil || len(phones) == 0 {
		ctx.JSON(400, gin.H{"success": false, "message": "Gagal mendapatkan nomor telepon dari WABA: " + req.WabaID})
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
		log.Printf("[WA Session Connect] gagal subscribe webhook WABA=%s: %v", req.WabaID, err)
	}

	// Simpan ke DB
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
		ctx.JSON(500, gin.H{"success": false, "message": "Gagal menyimpan koneksi"})
		return
	}
	cont.syncToTenantSettings(tenantSchema, phoneNumberID, accessToken)
	if pin := cont.ensureTenantRegistrationPin(tenantSchema); pin != "" && phoneNumberID != "" {
		waClient := helpers.NewWhatsAppClient(phoneNumberID, accessToken)
		registrationStatus := waRegistrationStatus{Ready: true}
		if err := waClient.RegisterPhoneNumber(pin); err != nil {
			log.Printf("[WA Session Connect] gagal register phone_number_id=%s: %v", phoneNumberID, err)
			registrationStatus.Ready = false
			if helpers.IsWhatsAppRegistrationBlocked(err) {
				registrationStatus.Message = "Connected, but this number is still attached to another WhatsApp account and cannot send messages yet."
			} else {
				registrationStatus.Message = "Connected, but the WhatsApp business number is not ready to send messages yet."
			}
		} else {
			log.Printf("[WA Session Connect] nomor WhatsApp berhasil diregister phone_number_id=%s", phoneNumberID)
		}
		cont.updateRegistrationStatus(tenantSchema, registrationStatus)
	}

	log.Printf("[WA Session Connect] ✅ terhubung schema=%s phone=%s waba=%s", tenantSchema, phoneNumber, req.WabaID)

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
		ctx.JSON(401, gin.H{"success": false, "message": "Token tidak valid atau sudah kadaluarsa"})
		return
	}

	userIDStr, _ := claims["user_id"].(string)
	tenantSchema, _ := claims["tenant_schema"].(string)
	if userIDStr == "" || tenantSchema == "" {
		ctx.JSON(401, gin.H{"success": false, "message": "Token tidak mengandung user context"})
		return
	}

	appID := os.Getenv("META_APP_ID")
	callbackURL := os.Getenv("META_OAUTH_CALLBACK_URL")
	if appID == "" || callbackURL == "" {
		ctx.JSON(500, gin.H{"success": false, "message": "META_APP_ID atau META_OAUTH_CALLBACK_URL belum dikonfigurasi"})
		return
	}

	stateData := map[string]interface{}{
		"user_id":       userIDStr,
		"tenant_schema": tenantSchema,
	}
	stateToken, err := helpers.GenerateJWT(cont.JwtKey, 10*time.Minute, stateData)
	if err != nil {
		ctx.JSON(500, gin.H{"success": false, "message": "Gagal generate state token"})
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

// OAuthCallback menerima redirect dari Meta setelah user login & grant permission.
// Backend otomatis: tukar code → extend token → cari WABA → ambil nomor → simpan → redirect FE.
// GET /api/v1/webhook/whatsapp/oauth-callback (public, no auth)
func (cont *WhatsAppOAuthContImpl) OAuthCallback(ctx *gin.Context) {
	frontendURL := os.Getenv("META_FRONTEND_REDIRECT_URL")
	if frontendURL == "" {
		frontendURL = "/"
	}

	redirectError := func(msg string) {
		ctx.Redirect(302, frontendURL+"?wa_status=error&message="+url.QueryEscape(msg))
	}

	// 1. Ambil code + state dari query
	code := ctx.Query("code")
	stateToken := ctx.Query("state")
	if code == "" || stateToken == "" {
		redirectError("Parameter tidak lengkap dari Meta")
		return
	}

	// 2. Validasi state token → pastikan request sah & dapat user context
	stateData, err := helpers.DecodeJWT(stateToken, cont.JwtKey)
	if err != nil {
		redirectError("State token tidak valid atau sudah kadaluarsa")
		return
	}
	userIDStr, _ := stateData["user_id"].(string)
	tenantSchema, _ := stateData["tenant_schema"].(string)
	if userIDStr == "" || tenantSchema == "" {
		redirectError("State token tidak mengandung data yang valid")
		return
	}

	callbackURL := os.Getenv("META_OAUTH_CALLBACK_URL")

	// 3. Tukar code dengan access token
	accessToken, err := helpers.ExchangeCodeForTokenWithURI(code, callbackURL)
	if err != nil {
		log.Printf("[WA Callback] gagal exchange code untuk user=%s: %v", userIDStr, err)
		redirectError("Gagal menghubungkan akun WhatsApp")
		return
	}

	// 4. Extend token agar tahan ~60 hari
	extendedToken, err := helpers.ExtendToken(accessToken)
	if err != nil {
		log.Printf("[WA Callback] gagal extend token, pakai token pendek: %v", err)
		extendedToken = accessToken
	}

	// 5. Ambil daftar WABA milik user → otomatis pilih yang pertama
	wabas, err := helpers.GetUserWABAs(extendedToken)
	if err != nil || len(wabas) == 0 {
		log.Printf("[WA Callback] WABA tidak ditemukan otomatis untuk user=%s, redirect ke manual input: %v", userIDStr, err)

		// Bungkus access_token + user context dalam short-lived JWT (15 menit)
		// agar HTML bisa kirim ke /connect tanpa expose raw token di URL
		sessionData := map[string]interface{}{
			"access_token":  extendedToken,
			"user_id":       userIDStr,
			"tenant_schema": tenantSchema,
		}
		sessionToken, err := helpers.GenerateJWT(cont.JwtKey, 15*time.Minute, sessionData)
		if err != nil {
			redirectError("Gagal generate session token")
			return
		}
		ctx.Redirect(302, frontendURL+
			"?wa_status=need_waba"+
			"&session="+url.QueryEscape(sessionToken))
		return
	}
	wabaID := wabas[0].ID

	// 6. Ambil nomor telepon dari WABA
	phones, err := helpers.GetWABAPhoneNumbers(wabaID, extendedToken)
	if err != nil || len(phones) == 0 {
		log.Printf("[WA Callback] tidak ada phone number di WABA=%s: %v", wabaID, err)
		redirectError("Tidak ditemukan nomor telepon di akun WhatsApp Business")
		return
	}
	phoneNumberID := phones[0].ID
	phoneNumber := phones[0].DisplayPhoneNumber
	displayName := phones[0].VerifiedName

	// 7. Subscribe app ke webhook events WABA (non-fatal jika gagal)
	if err := helpers.SubscribeAppToWABA(wabaID, extendedToken); err != nil {
		log.Printf("[WA Callback] peringatan: gagal subscribe webhook WABA=%s: %v", wabaID, err)
	}

	// 8. Simpan koneksi ke DB
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
		log.Printf("[WA Callback] gagal simpan koneksi schema=%s: %v", tenantSchema, err)
		redirectError("Gagal menyimpan koneksi")
		return
	}

	// 9. Sync ke tenant settings agar getWhatsAppClient() langsung bisa pakai
	cont.syncToTenantSettings(tenantSchema, phoneNumberID, extendedToken)
	if pin := cont.ensureTenantRegistrationPin(tenantSchema); pin != "" && phoneNumberID != "" {
		waClient := helpers.NewWhatsAppClient(phoneNumberID, extendedToken)
		registrationStatus := waRegistrationStatus{Ready: true}
		if err := waClient.RegisterPhoneNumber(pin); err != nil {
			log.Printf("[WA Callback] gagal register phone_number_id=%s: %v", phoneNumberID, err)
			registrationStatus.Ready = false
			if helpers.IsWhatsAppRegistrationBlocked(err) {
				registrationStatus.Message = "Connected, but this number is still attached to another WhatsApp account and cannot send messages yet."
			} else {
				registrationStatus.Message = "Connected, but the WhatsApp business number is not ready to send messages yet."
			}
		} else {
			log.Printf("[WA Callback] nomor WhatsApp berhasil diregister phone_number_id=%s", phoneNumberID)
		}
		cont.updateRegistrationStatus(tenantSchema, registrationStatus)
	}

	log.Printf("[WA Callback] ✅ terhubung schema=%s phone=%s waba=%s", tenantSchema, phoneNumber, wabaID)

	// 10. Redirect ke FE dengan info sukses
	ctx.Redirect(302, frontendURL+
		"?wa_status=connected"+
		"&phone="+url.QueryEscape(phoneNumber)+
		"&display_name="+url.QueryEscape(displayName))
}

// getUserContext mengambil user_id dan tenant_schema dari JWT
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

// syncToTenantSettings menyinkronkan credentials ke tenant setting table
// agar logika getWhatsAppClient() yang sudah ada tetap berfungsi
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
			log.Printf("[WA OAuth] gagal sync setting %s ke schema=%s: %v", s.name, schema, err)
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
		log.Printf("[WA OAuth] gagal simpan registration pin schema=%s: %v", schema, err)
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
			log.Printf("[WA OAuth] gagal sync status %s ke schema=%s: %v", s.name, schema, err)
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

// clearTenantSettings menghapus credentials WhatsApp dari tenant setting table
func (cont *WhatsAppOAuthContImpl) clearTenantSettings(schema string) {
	err := cont.Db.Exec(
		`UPDATE `+schema+`.setting SET value = '', updated_at = NOW()
		WHERE sub_group_name = 'WhatsApp' AND name IN ('whatsapp-phone-number-id', 'whatsapp-access-token', 'whatsapp-registration-pin', 'whatsapp-ready-to-send', 'whatsapp-status-message')`,
	).Error
	if err != nil {
		log.Printf("[WA OAuth] gagal clear settings untuk schema=%s: %v", schema, err)
	}
}
