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

	settings := req.UpdateSettingItemsToSettings(subGroupName, requests.Settings)

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

	// Update setting
	setting := domains.Setting{
		GroupName:    "integration",
		SubgroupName: "Telegram",
		Name:         "telegram-bot-token",
		Value:        botToken,
	}

	err = serv.SettingRepo.Create(serv.Db, schema, setting)
	if err != nil {
		return fmt.Errorf("failed to update setting: %w", err)
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
