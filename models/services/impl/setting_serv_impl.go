package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	"backend/models/repositories/impl"
	req "backend/models/requests/setting"
	"backend/models/responses/setting"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	stripewebhookendpoint "github.com/stripe/stripe-go/v81/webhookendpoint"
	"gorm.io/gorm"
)

type SettingServImpl struct {
	Db          *gorm.DB
	JwtKey      string
	SettingRepo repositories.SettingRepo
}

func NewSettingServImpl(db *gorm.DB, jwtKey string, settingRepo repositories.SettingRepo) *SettingServImpl {
	return &SettingServImpl{Db: db, JwtKey: jwtKey, SettingRepo: settingRepo}
}

func (serv *SettingServImpl) getSchema(accessToken string, role *string) (string, error) {
	if *role == "SuperAdmin" || *role == "Admin" {
		return "public", nil
	}
	schema, err := helpers.GetUsernameFromToken(accessToken, serv.JwtKey)
	if err != nil {
		return "", err
	}
	return *schema, nil
}

func (serv *SettingServImpl) GetNotification(accessToken string) (*setting.GroupResponse, error) {
	role, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Admin", "Client"})
	if err != nil || !ok {
		return nil, err
	}

	schema, err := serv.getSchema(accessToken, role)
	if err != nil {
		return nil, err
	}

	result, errResult := serv.SettingRepo.GetByGroupName(serv.Db, schema, "notification")
	if errResult != nil {
		log.Printf("[SettingRepo].GetByGroupName error: %v", errResult)
		return nil, fmt.Errorf("failed to get setting notification")
	}

	response := setting.ToGroupResponse(result)
	return response, nil
}

func (serv *SettingServImpl) GetIntegration(accessToken string) (*setting.GroupResponse, error) {
	role, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Admin", "Client"})
	if err != nil || !ok {
		return nil, err
	}

	schema, err := serv.getSchema(accessToken, role)
	if err != nil {
		return nil, err
	}

	var result []domains.Setting
	var errResult error

	if *role == "Client" {
		result, errResult = serv.SettingRepo.GetByGroupAndSubGroupName(serv.Db, schema, "integration", "Telegram")
	} else {
		result, errResult = serv.SettingRepo.GetByGroupName(serv.Db, schema, "integration")
	}

	if errResult != nil {
		log.Printf("[SettingRepo].GetIntegration error: %v", errResult)
		return nil, fmt.Errorf("failed to get setting integration")
	}

	response := setting.ToGroupResponse(result)
	return response, nil
}

func (serv *SettingServImpl) UpdateBySubgroupName(accessToken, subGroupName string, requests req.UpdateBySubgroupRequest) error {
	role, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Admin", "Client"})
	if err != nil || !ok {
		return err
	}

	schema, err := serv.getSchema(accessToken, role)
	if err != nil {
		return err
	}

	settings := requests.ToSettings(subGroupName)

	if err := serv.SettingRepo.UpdateBySubGroupName(serv.Db, schema, settings); err != nil {
		log.Printf("[SettingRepo].UpdateBySubGroupName error: %v", err)
		return fmt.Errorf("failed to update setting")
	}

	// Auto-register webhook Stripe kalau client update Stripe Client secret key
	if *role == "Client" && subGroupName == "Stripe Client" {
		secretKey := ""
		for _, s := range settings {
			if s.Name == "stripe-client-secret-key" {
				secretKey = s.Value
				break
			}
		}

		if secretKey != "" {
			// Hapus webhook lama dulu sebelum register baru
			serv.deleteExistingWebhook(secretKey, schema)

			webhookSecret, err := serv.registerStripeWebhook(secretKey, schema)
			if err != nil {
				// Log error tapi tidak fail — secret key sudah tersimpan
				log.Printf("[SettingServ] registerStripeWebhook error: %v", err)
			} else {
				webhookSettings := []domains.Setting{
					{
						SubgroupName: "Stripe Client",
						Name:         "stripe-client-webhook-secret",
						Value:        webhookSecret,
					},
				}
				if err := serv.SettingRepo.UpdateBySubGroupName(serv.Db, schema, webhookSettings); err != nil {
					log.Printf("[SettingRepo] save webhook secret error: %v", err)
				} else {
					log.Printf("[SettingServ] webhook registered and secret saved for schema: %s", schema)
				}
			}
		}
	}

	return nil
}

func (serv *SettingServImpl) deleteExistingWebhook(secretKey, schema string) {
	stripe.Key = secretKey

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		log.Printf("[SettingServ] APP_URL not set, skip delete existing webhook")
		return
	}

	webhookURL := fmt.Sprintf("%s/api/v1/payments/client/webhook/%s", appURL, schema)

	params := &stripe.WebhookEndpointListParams{}
	iter := stripewebhookendpoint.List(params)
	for iter.Next() {
		ep := iter.WebhookEndpoint()
		if ep.URL == webhookURL {
			_, err := stripewebhookendpoint.Del(ep.ID, nil)
			if err != nil {
				log.Printf("[SettingServ] failed to delete old webhook %s: %v", ep.ID, err)
			} else {
				log.Printf("[SettingServ] deleted old webhook: %s", ep.ID)
			}
		}
	}

	if err := iter.Err(); err != nil {
		log.Printf("[SettingServ] list webhook error: %v", err)
	}
}

func (serv *SettingServImpl) registerStripeWebhook(secretKey, schema string) (string, error) {
	stripe.Key = secretKey

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		return "", fmt.Errorf("APP_URL env not set")
	}

	webhookURL := fmt.Sprintf("%s/api/v1/payments/client/webhook/%s", appURL, schema)

	params := &stripe.WebhookEndpointParams{
		URL: stripe.String(webhookURL),
		EnabledEvents: []*string{
			stripe.String("invoice.paid"),
			stripe.String("invoice.payment_failed"),
		},
		Description: stripe.String(fmt.Sprintf("Webhook for schema: %s", schema)),
	}

	endpoint, err := stripewebhookendpoint.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to register stripe webhook: %w", err)
	}

	log.Printf("[SettingServ] stripe webhook registered: %s, url: %s", endpoint.ID, webhookURL)

	return endpoint.Secret, nil
}

// UpdateTelegramBotToken updates Telegram bot token and registers webhook
func (serv *SettingServImpl) UpdateTelegramBotToken(accessToken string, clientID uuid.UUID, botToken string) error {
	// Validate token
	_, err := helpers.DecodeJWT(accessToken, serv.JwtKey)
	if err != nil {
		return fmt.Errorf("invalid token")
	}

	// Get user to find schema
	userRepo := impl.NewUserRepoImpl()
	user, err := userRepo.GetByUserId(serv.Db, clientID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	if user.TenantSchema == nil || *user.TenantSchema == "" {
		return fmt.Errorf("tenant schema not found")
	}

	schema := helpers.NormalizeSchema(*user.TenantSchema)

	// Check if setting already exists
	existingSettings, err := serv.SettingRepo.GetByGroupAndSubGroupName(serv.Db, schema, "integration", "Telegram")
	if err != nil {
		return fmt.Errorf("failed to get existing settings: %w", err)
	}

	// Update or create setting
	setting := domains.Setting{
		GroupName:    "integration",
		SubgroupName: "Telegram",
		Name:         "telegram-bot-token",
		Value:        botToken,
	}

	if len(existingSettings) > 0 {
		// Update existing setting
		for i, s := range existingSettings {
			if s.Name == "telegram-bot-token" {
				existingSettings[i].Value = botToken
				if err := serv.SettingRepo.UpdateBySubGroupName(serv.Db, schema, existingSettings); err != nil {
					return fmt.Errorf("failed to update setting: %w", err)
				}
				break
			}
		}
	} else {
		// Create new setting
		if err := serv.SettingRepo.Create(serv.Db, schema, setting); err != nil {
			return fmt.Errorf("failed to create setting: %w", err)
		}
	}

	// Register webhook to Telegram API
	webhookURL := fmt.Sprintf("%s/api/v1/webhook/telegram/%s", os.Getenv("APP_URL"), schema)
	tgClient := helpers.NewTelegramClient(botToken)
	_, err = tgClient.SetWebhook(webhookURL)
	if err != nil {
		return fmt.Errorf("failed to register Telegram webhook: %w", err)
	}

	log.Printf("[SettingServ] Telegram webhook registered: %s", webhookURL)

	return nil
}

func (serv *SettingServImpl) GetJwtKey() string {
	return serv.JwtKey
}

func (serv *SettingServImpl) GetDb() *gorm.DB {
	return serv.Db
}

func (serv *SettingServImpl) GetUserRepo() repositories.UsersRepo {
	// Return nil - not used in SettingServ
	return nil
}

func (serv *SettingServImpl) GetByGroupAndSubGroupName(db *gorm.DB, schema, group, subGroup string) ([]interface{}, error) {
	settings, err := serv.SettingRepo.GetByGroupAndSubGroupName(db, schema, group, subGroup)
	if err != nil {
		return nil, err
	}
	
	// Convert to interface{} slice
	result := make([]interface{}, len(settings))
	for i, s := range settings {
		result[i] = map[string]interface{}{
			"name":  s.Name,
			"value": s.Value,
		}
	}
	return result, nil
}

func (serv *SettingServImpl) UpdateBySubGroupNameForSchema(db *gorm.DB, schema, subGroupName, name, value string) error {
	// Direct SQL update
	return db.Table(schema + ".setting").
		Where("sub_group_name = ? AND name = ?", subGroupName, name).
		Update("value", value).Error
}

// aiPromptSectionMap maps API section name to (sub_group_name, setting name)
var aiPromptSectionMap = map[string][2]string{
	"product":     {"AI Product", "ai-product-prompt"},
	"delivery":    {"AI Delivery", "ai-delivery-prompt"},
	"operational": {"AI Operational", "ai-operational-prompt"},
	"about-store": {"AI About Store", "ai-about-store-prompt"},
	"faq":         {"AI FAQ", "ai-faq-prompt"},
}

// GetAIPrompts returns all 4 AI prompt sections for a tenant schema (no auth required — used internally)
func (serv *SettingServImpl) GetAIPrompts(schema string) (map[string]string, error) {
	result := map[string]string{}
	for section, meta := range aiPromptSectionMap {
		settings, err := serv.SettingRepo.GetByGroupAndSubGroupName(serv.Db, schema, "ai_prompt", meta[0])
		if err != nil {
			result[section] = ""
			continue
		}
		val := ""
		for _, s := range settings {
			if s.Name == meta[1] {
				val = s.Value
				break
			}
		}
		result[section] = val
	}
	return result, nil
}

// UpdateAIPromptSection upserts a single AI prompt section for a tenant schema
func (serv *SettingServImpl) UpdateAIPromptSection(accessToken, schema, section, prompt string) error {
	_, err := helpers.DecodeJWT(accessToken, serv.JwtKey)
	if err != nil {
		return fmt.Errorf("invalid token")
	}

	meta, ok := aiPromptSectionMap[section]
	if !ok {
		return fmt.Errorf("invalid section '%s': valid values are product, delivery, about-store, faq", section)
	}

	existing, err := serv.SettingRepo.GetByGroupAndSubGroupName(serv.Db, schema, "ai_prompt", meta[0])
	if err != nil {
		return fmt.Errorf("failed to query settings: %w", err)
	}

	found := false
	for _, s := range existing {
		if s.Name == meta[1] {
			found = true
			break
		}
	}

	if found {
		return serv.SettingRepo.UpdateBySubGroupName(serv.Db, schema, []domains.Setting{
			{GroupName: "ai_prompt", SubgroupName: meta[0], Name: meta[1], Value: prompt},
		})
	}

	return serv.SettingRepo.Create(serv.Db, schema, domains.Setting{
		GroupName:    "ai_prompt",
		SubgroupName: meta[0],
		Name:         meta[1],
		Value:        prompt,
	})
}

