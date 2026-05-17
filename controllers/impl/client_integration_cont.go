package impl

import (
	"backend/exceptions"
	"backend/helpers"
	req "backend/models/requests/setting"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v81"
	stripeAccount "github.com/stripe/stripe-go/v81/account"
)

// GetClientIntegration godoc
// @Summary      Get Client Integration Settings
// @Description  Get all integration settings for client (Telegram, Stripe, etc.)
// @Tags         Settings
// @Produce      json
// @Param        client_id  path  string  true  "Client ID"
// @Success      200        {object}  helpers.ApiResponse{data=[]setting.SubgroupResponse}
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      404        {object}  helpers.ApiResponse
// @Security     BearerAuth
// @Router       /client/{client_id}/integration [get]
func (cont *SettingContImpl) GetClientIntegration(context *gin.Context) {
	clientID, err := helpers.ParseUUID(context, "client_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	// Get schema from client_id
	schema, err := helpers.GetSchema(cont.Db, cont.UserRepo, clientID)
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	// Get all settings from integration group - query each subgroup
	subgroups := []string{"Telegram", "WhatsApp", "Stripe Client"}
	responseData := []map[string]interface{}{}

	for _, subGroupName := range subgroups {
		settings, err := cont.SettingServ.GetByGroupAndSubGroupName(cont.Db, schema, "integration", subGroupName)
		if err != nil {
			continue // Skip if error
		}

		settingsData := []map[string]string{}
		for _, s := range settings {
			setting := s.(map[string]interface{})
			name, _ := setting["name"].(string)
			value, _ := setting["value"].(string)

			// Include ALL settings (including sensitive ones)
			// FE needs to know all fields exist for proper updates
			settingsData = append(settingsData, map[string]string{
				"name":  name,
				"value": value,
			})
		}

		if len(settingsData) > 0 {
			responseData = append(responseData, map[string]interface{}{
				"sub_group_name": subGroupName,
				"settings":       settingsData,
			})
		}
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    responseData,
	}

	errResponse := helpers.WriteToResponseBody(context, response.Code, response)
	if errResponse != nil {
		exceptions.ErrorHandler(context, errResponse)
		return
	}
}

// UpdateClientIntegration godoc
// @Summary      Update Client Integration Settings by Subgroup
// @Description  Update all settings in a subgroup for client (Telegram, Stripe, etc.)
// @Tags         Settings
// @Accept       json
// @Produce      json
// @Param        client_id  path  string  true  "Client ID"
// @Param        sub_group_name  path  string  true  "Subgroup Name (e.g., Telegram, Stripe Client)"
// @Param        request    body  setting.UpdateBySubgroupRequest  true  "Settings to update"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      404        {object}  helpers.ApiResponse
// @Security     BearerAuth
// @Router       /client/{client_id}/integration/{sub_group_name} [patch]
func (cont *SettingContImpl) UpdateClientIntegration(context *gin.Context) {
	clientID, err := helpers.ParseUUID(context, "client_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	subGroupName := context.Param("sub_group_name")
	if subGroupName == "" {
		exceptions.ErrorHandler(context, fmt.Errorf("sub_group_name is required"))
		return
	}

	var request req.UpdateBySubgroupRequest

	if err := helpers.ReadFromRequestBody(context, &request); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	// Get schema from client_id
	schema, err := helpers.GetSchema(cont.Db, cont.UserRepo, clientID)
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	// Block toggle to AI mode when outside operational hours
	if wantsAIMode(request.Settings) {
		if err := cont.enforceOperationalHours(schema); err != nil {
			exceptions.ErrorHandler(context, err)
			return
		}
	}

	// Update all settings in transaction
	tx := cont.Db.Begin()
	if tx.Error != nil {
		exceptions.ErrorHandler(context, fmt.Errorf("failed to start transaction"))
		return
	}

	for _, s := range request.Settings {
		err := cont.SettingServ.UpdateBySubGroupNameForSchema(tx, schema, "integration", subGroupName, s.Name, s.Value)
		if err != nil {
			tx.Rollback()
			exceptions.ErrorHandler(context, fmt.Errorf("failed to update setting %s: %w", s.Name, err))
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		exceptions.ErrorHandler(context, fmt.Errorf("failed to commit transaction: %w", err))
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data: map[string]string{
			"message": fmt.Sprintf("All settings updated for subgroup: %s", subGroupName),
		},
	}

	errResponse := helpers.WriteToResponseBody(context, response.Code, response)
	if errResponse != nil {
		exceptions.ErrorHandler(context, errResponse)
		return
	}
}

// wantsAIMode returns true when any setting in the batch is trying to switch to AI mode.
func wantsAIMode(settings []req.UpdateSettingItem) bool {
	for _, s := range settings {
		if s.Name == "manual-mode" && s.Value == "false" {
			return true
		}
		if s.Name == "bot-enabled" && s.Value == "true" {
			return true
		}
	}
	return false
}

// enforceOperationalHours returns an error when the current time is outside the configured
// AI operational hours for the given schema, blocking any toggle-to-AI request.
func (cont *SettingContImpl) enforceOperationalHours(schema string) error {
	var hoursJSON string
	cont.Db.Raw(fmt.Sprintf(
		`SELECT value FROM %s.setting WHERE group_name = 'ai_prompt' AND sub_group_name = 'AI Operational' AND name = 'ai-operational-prompt' LIMIT 1`,
		schema,
	)).Scan(&hoursJSON)

	if hoursJSON == "" {
		cont.Db.Raw(fmt.Sprintf(
			`SELECT value FROM %s.setting WHERE group_name = 'ai_prompt' AND sub_group_name = 'Store Operational' AND name = 'store-operational-hours' LIMIT 1`,
			schema,
		)).Scan(&hoursJSON)
	}

	// No schedule configured — always allow
	if hoursJSON == "" {
		return nil
	}

	var timezone string
	cont.Db.Raw(fmt.Sprintf(
		`SELECT value FROM %s.setting WHERE group_name = 'integration' AND sub_group_name = 'Telegram' AND name = 'timezone' LIMIT 1`,
		schema,
	)).Scan(&timezone)
	if timezone == "" {
		timezone = "Asia/Singapore"
	}

	withinHours, _ := helpers.IsWithinOperationalHours(hoursJSON, timezone)
	if !withinHours {
		return fmt.Errorf("outside_operational_hours")
	}
	return nil
}

// GetClientStripeStatus godoc
// @Summary      Get Client Stripe Connection Status
// @Description  Validates the tenant's stored Stripe secret key by making a live API call. Returns connected=true only when the key is present and accepted by Stripe.
// @Tags         Settings
// @Produce      json
// @Param        client_id  path  string  true  "Client ID"
// @Success      200  {object}  helpers.ApiResponse{data=map[string]bool}
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      401  {object}  helpers.ApiResponse
// @Security     BearerAuth
// @Router       /client/{client_id}/integration/stripe/status [get]
func (cont *SettingContImpl) GetClientStripeStatus(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	schema, err := helpers.GetSchema(cont.Db, cont.UserRepo, clientID)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	var secretKey string
	cont.Db.Raw(fmt.Sprintf(
		`SELECT value FROM %s.setting WHERE group_name = 'integration' AND sub_group_name = 'Stripe Client' AND name = 'stripe-client-secret-key' LIMIT 1`,
		schema,
	)).Scan(&secretKey)

	connected := false
	if secretKey != "" {
		stripe.Key = secretKey
		_, stripeErr := stripeAccount.Get()
		connected = stripeErr == nil
	}

	errResponse := helpers.WriteToResponseBody(ctx, http.StatusOK, helpers.ApiResponse{
		Success: true,
		Code:    http.StatusOK,
		Data:    map[string]bool{"connected": connected},
	})
	if errResponse != nil {
		exceptions.ErrorHandler(ctx, errResponse)
	}
}
