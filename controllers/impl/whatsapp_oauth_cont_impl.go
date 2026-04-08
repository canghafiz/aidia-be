package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WhatsAppOAuthContImpl struct {
	WhatsAppConnectionRepo repositories.WhatsAppConnectionRepo
	SettingRepo            repositories.SettingRepo
	Db                     *gorm.DB
	JwtKey                 string
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
	Code          string `json:"code" binding:"required"`
	WabaID        string `json:"waba_id" binding:"required"`
	PhoneNumberID string `json:"phone_number_id"` // opsional, jika sudah diketahui dari Embedded Signup
}

// Connect menghubungkan WhatsApp Business akun tenant via Meta Embedded Signup.
// POST /api/v1/client/:client_id/whatsapp/connect
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

	// 1. Tukar code dengan access token
	log.Printf("[WA OAuth] menukar code untuk user=%s", userIDStr)
	accessToken, err := helpers.ExchangeCodeForToken(req.Code)
	if err != nil {
		log.Printf("[WA OAuth] gagal exchange code: %v", err)
		ctx.JSON(400, gin.H{"success": false, "message": "Gagal menghubungkan akun WhatsApp: " + err.Error()})
		return
	}

	// 2. Extend token agar tahan ~60 hari
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
		// Non-fatal: log saja, koneksi tetap disimpan
		log.Printf("[WA OAuth] peringatan: gagal subscribe webhook ke WABA: %v", err)
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
		},
	})
}

// Status mengecek status koneksi WhatsApp tenant.
// GET /api/v1/client/:client_id/whatsapp/status
func (cont *WhatsAppOAuthContImpl) Status(ctx *gin.Context) {
	userIDStr, _ := cont.getUserContext(ctx)
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
		},
	})
}

// Disconnect memutuskan koneksi WhatsApp tenant.
// DELETE /api/v1/client/:client_id/whatsapp/disconnect
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
			ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			s.name, s.value,
		).Error
		if err != nil {
			log.Printf("[WA OAuth] gagal sync setting %s ke schema=%s: %v", s.name, schema, err)
		}
	}
}

// clearTenantSettings menghapus credentials WhatsApp dari tenant setting table
func (cont *WhatsAppOAuthContImpl) clearTenantSettings(schema string) {
	err := cont.Db.Exec(
		`UPDATE `+schema+`.setting SET value = '', updated_at = NOW()
		WHERE sub_group_name = 'WhatsApp' AND name IN ('whatsapp-phone-number-id', 'whatsapp-access-token')`,
	).Error
	if err != nil {
		log.Printf("[WA OAuth] gagal clear settings untuk schema=%s: %v", schema, err)
	}
}
