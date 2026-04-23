package impl

import (
	"backend/exceptions"
	"backend/helpers"
	req "backend/models/requests/setting"
	"fmt"

	"github.com/gin-gonic/gin"
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
	subgroups := []string{"Telegram", "Stripe Client", "HitPay Client"}
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

	// Update all settings in transaction
	tx := cont.Db.Begin()
	if tx.Error != nil {
		exceptions.ErrorHandler(context, fmt.Errorf("failed to start transaction"))
		return
	}

	for _, s := range request.Settings {
		err := cont.SettingServ.UpdateBySubGroupNameForSchema(tx, schema, subGroupName, s.Name, s.Value)
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
