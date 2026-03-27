package impl

import (
	"backend/exceptions"
	"backend/helpers"
	"fmt"
	req "backend/models/requests/setting"
	"backend/models/services"

	"github.com/gin-gonic/gin"
)

type SettingContImpl struct {
	SettingServ services.SettingServ
}

func NewSettingContImpl(settingServ services.SettingServ) *SettingContImpl {
	return &SettingContImpl{SettingServ: settingServ}
}

// GetNotification godoc
// @Summary      Get Notification Settings
// @Description  Ambil data setting notifikasi
// @Tags         Settings
// @Produce      json
// @Success      200  {object}  helpers.ApiResponse{data=setting.GroupResponse}
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      401  {object}  helpers.ApiResponse
// @Security     BearerAuth
// @Router       /settings/notification [get]
func (cont *SettingContImpl) GetNotification(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	result, err := cont.SettingServ.GetNotification(jwtToken)
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    result,
	}

	errResponse := helpers.WriteToResponseBody(context, response.Code, response)
	if errResponse != nil {
		exceptions.ErrorHandler(context, errResponse)
		return
	}
}

// GetIntegration godoc
// @Summary      Get Integration Settings
// @Description  Ambil data setting integrasi. SuperAdmin mendapat semua data, Client hanya mendapat data Telegram
// @Tags         Settings
// @Produce      json
// @Success      200  {object}  helpers.ApiResponse{data=setting.GroupResponse}
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      401  {object}  helpers.ApiResponse
// @Security     BearerAuth
// @Router       /settings/integration [get]
func (cont *SettingContImpl) GetIntegration(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	result, err := cont.SettingServ.GetIntegration(jwtToken)
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    result,
	}

	errResponse := helpers.WriteToResponseBody(context, response.Code, response)
	if errResponse != nil {
		exceptions.ErrorHandler(context, errResponse)
		return
	}
}

// UpdateBySubgroupName godoc
// @Summary      Update Settings by Subgroup
// @Description  Update value setting berdasarkan subgroup name
// @Tags         Settings
// @Accept       json
// @Produce      json
// @Param        sub_group_name  path      string                      true  "Subgroup Name"
// @Param        request         body      req.UpdateBySubgroupRequest  true  "Update Setting Request"
// @Success      200             {object}  helpers.ApiResponse
// @Failure      400             {object}  helpers.ApiResponse
// @Failure      401             {object}  helpers.ApiResponse
// @Security     BearerAuth
// @Router       /settings/subgroup-name/{sub_group_name} [patch]
func (cont *SettingContImpl) UpdateBySubgroupName(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)
	subGroupName := context.Param("sub_group_name")

	request := req.UpdateBySubgroupRequest{}
	errParse := helpers.ReadFromRequestBody(context, &request)
	if errParse != nil {
		exceptions.ErrorHandler(context, errParse)
		return
	}

	err := cont.SettingServ.UpdateBySubgroupName(jwtToken, subGroupName, request)
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    nil,
	}

	errResponse := helpers.WriteToResponseBody(context, response.Code, response)
	if errResponse != nil {
		exceptions.ErrorHandler(context, errResponse)
		return
	}
}

// UpdateTelegramBotToken godoc
// @Summary      Update Telegram Bot Token
// @Description  Update Telegram bot token and auto-register webhook
// @Tags         Settings
// @Accept       json
// @Produce      json
// @Param        client_id  path  string  true  "Client ID"
// @Param        request    body  object  true  "Bot token request"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Security     BearerAuth
// @Router       /client/{client_id}/telegram/bot-token [patch]
func (cont *SettingContImpl) UpdateTelegramBotToken(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	clientID, err := helpers.ParseUUID(context, "client_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	var request struct {
		BotToken string `json:"bot_token" validate:"required"`
	}

	if err := helpers.ReadFromRequestBody(context, &request); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	if request.BotToken == "" {
		exceptions.ErrorHandler(context, fmt.Errorf("bot_token is required"))
		return
	}

	// Update setting dan register webhook
	err = cont.SettingServ.UpdateTelegramBotToken(jwtToken, clientID, request.BotToken)
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    map[string]string{"message": "Telegram bot token updated and webhook registered"},
	}

	errResponse := helpers.WriteToResponseBody(context, response.Code, response)
	if errResponse != nil {
		exceptions.ErrorHandler(context, errResponse)
		return
	}
}
