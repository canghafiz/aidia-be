package impl

import (
	"backend/exceptions"
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	req "backend/models/requests/setting"
	"backend/models/services"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	pdf "github.com/ledongthuc/pdf"
	"gorm.io/gorm"
)

type SettingContImpl struct {
	SettingServ       services.SettingServ
	UserRepo          repositories.UsersRepo
	Db                *gorm.DB
	FileServ          services.FileServ
	KnowledgeBaseRepo repositories.KnowledgeBaseRepo
}

func NewSettingContImpl(settingServ services.SettingServ, userRepo repositories.UsersRepo, db *gorm.DB, fileServ services.FileServ, kbRepo repositories.KnowledgeBaseRepo) *SettingContImpl {
	return &SettingContImpl{
		SettingServ:       settingServ,
		UserRepo:          userRepo,
		Db:                db,
		FileServ:          fileServ,
		KnowledgeBaseRepo: kbRepo,
	}
}

// GetNotification godoc
// @Summary      Get Notification Settings
// @Description  Get notification settings data
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
// @Description  Get integration settings data. SuperAdmin receives all data; Client only receives Telegram data
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
// @Description  Update setting values by subgroup name
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
	subGroupName := context.Query("sub_group_name")

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
// @Param        request    body  telegram.UpdateBotTokenRequest  true  "Update Bot Token Request"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
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

	// Update setting and register webhook
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

// GetClientAIPrompts godoc
// @Summary      Get All AI Prompt Sections
// @Description  Get all 5 AI prompt sections (product, delivery, operational, about-store, faq) for a tenant
// @Tags         AI Prompts
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true  "Client ID"
// @Success      200  {object}  helpers.ApiResponse{data=req.AIPromptsResponse}
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      401  {object}  helpers.ApiResponse
// @Router       /client/{client_id}/settings/ai-prompts [get]
func (cont *SettingContImpl) GetClientAIPrompts(context *gin.Context) {
	clientID, err := helpers.ParseUUID(context, "client_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	schema, err := helpers.GetSchema(cont.Db, cont.UserRepo, clientID)
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	prompts, err := cont.SettingServ.GetAIPrompts(schema)
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	response := req.AIPromptsResponse{
		Product:          prompts["product"],
		Delivery:         prompts["delivery"],
		Operational:      prompts["operational"],
		AboutStore:       prompts["about-store"],
		Faq:              prompts["faq"],
		StoreOperational: prompts["store-operational"],
	}

	helpers.WriteToResponseBody(context, 200, helpers.ApiResponse{Success: true, Code: 200, Data: response})
}

// GetClientAIPromptSection godoc
// @Summary      Get AI Prompt by Section
// @Description  Get AI prompt for a specific section. Valid sections: product, delivery, operational, about-store, faq
// @Tags         AI Prompts
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true  "Client ID"
// @Param        section    path  string  true  "Section name"  Enums(product, delivery, operational, about-store, faq)
// @Success      200  {object}  helpers.ApiResponse{data=req.AIPromptSectionResponse}
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      401  {object}  helpers.ApiResponse
// @Router       /client/{client_id}/settings/ai-prompts/{section} [get]
func (cont *SettingContImpl) GetClientAIPromptSection(context *gin.Context) {
	clientID, err := helpers.ParseUUID(context, "client_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	section := context.Param("section")
	if section == "" {
		exceptions.ErrorHandler(context, fmt.Errorf("section is required"))
		return
	}

	schema, err := helpers.GetSchema(cont.Db, cont.UserRepo, clientID)
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	prompts, err := cont.SettingServ.GetAIPrompts(schema)
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	val, ok := prompts[section]
	if !ok {
		exceptions.ErrorHandler(context, fmt.Errorf("invalid section '%s': valid values are product, delivery, about-store, faq", section))
		return
	}

	helpers.WriteToResponseBody(context, 200, helpers.ApiResponse{
		Success: true, Code: 200,
		Data: map[string]string{"section": section, "prompt": val},
	})
}

// GetClientKDS godoc
// @Summary      Get KDS status for a client
// @Description  Returns whether Kitchen Display System is enabled for the given client. Accessible by SuperAdmin and the Client itself.
// @Tags         Settings
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true  "Client ID"
// @Success      200  {object}  helpers.ApiResponse{data=map[string]interface{}}
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      401  {object}  helpers.ApiResponse
// @Router       /client/{client_id}/settings/kds [get]
func (cont *SettingContImpl) GetClientKDS(ctx *gin.Context) {
	jwt := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	enabled, err := cont.SettingServ.GetClientKDS(jwt, clientID)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	helpers.WriteToResponseBody(ctx, 200, helpers.ApiResponse{
		Success: true, Code: 200,
		Data: map[string]interface{}{"kds_enabled": enabled, "client_id": clientID},
	})
}

// SetClientKDS godoc
// @Summary      Enable or disable KDS for a client
// @Description  SuperAdmin sets whether KDS is enabled for the given client. Body: {"enabled": true/false}
// @Tags         Settings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true  "Client ID"
// @Param        request    body  object  true  "KDS toggle"
// @Success      200  {object}  helpers.ApiResponse
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      401  {object}  helpers.ApiResponse
// @Router       /client/{client_id}/settings/kds [put]
func (cont *SettingContImpl) SetClientKDS(ctx *gin.Context) {
	jwt := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := helpers.ReadFromRequestBody(ctx, &body); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	if err := cont.SettingServ.SetClientKDS(jwt, clientID, body.Enabled); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	helpers.WriteToResponseBody(ctx, 200, helpers.ApiResponse{Success: true, Code: 200, Data: nil})
}

// UpdateClientAIPromptSection godoc
// @Summary      Update AI Prompt by Section
// @Description  Update AI prompt for a specific section. Valid sections: product, delivery, operational, about-store, faq
// @Tags         AI Prompts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string                    true  "Client ID"
// @Param        section    path  string                    true  "Section name"  Enums(product, delivery, operational, about-store, faq)
// @Param        request    body  req.UpdateAIPromptRequest  true  "Prompt content"
// @Success      200  {object}  helpers.ApiResponse
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      401  {object}  helpers.ApiResponse
// @Router       /client/{client_id}/settings/ai-prompts/{section} [patch]
func (cont *SettingContImpl) UpdateClientAIPromptSection(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	clientID, err := helpers.ParseUUID(context, "client_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	section := context.Param("section")
	if section == "" {
		exceptions.ErrorHandler(context, fmt.Errorf("section is required"))
		return
	}

	var request req.UpdateAIPromptRequest
	if err := helpers.ReadFromRequestBody(context, &request); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	schema, err := helpers.GetSchema(cont.Db, cont.UserRepo, clientID)
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	if err := cont.SettingServ.UpdateAIPromptSection(jwtToken, schema, section, request.Prompt); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	if section == "operational" {
		cont.syncBotStateFromOperationalHours(schema, request.Prompt)
	} else if section == "store-operational" {
		// Sync immediately only when AI Operational hours are not set.
		// If they are set, the scheduler will re-apply AI hours within a minute.
		var aiHours string
		cont.Db.Raw(fmt.Sprintf(
			`SELECT value FROM %s.setting WHERE group_name = 'ai_prompt' AND sub_group_name = 'AI Operational' AND name = 'ai-operational-prompt' LIMIT 1`,
			schema,
		)).Scan(&aiHours)
		if aiHours == "" {
			cont.syncBotStateFromOperationalHours(schema, request.Prompt)
		}
	}

	helpers.WriteToResponseBody(context, 200, helpers.ApiResponse{
		Success: true, Code: 200,
		Data: map[string]string{"message": fmt.Sprintf("AI prompt for section '%s' updated", section)},
	})
}

// syncBotStateFromOperationalHours immediately sets manual-mode and bot-enabled for
// both Telegram and WhatsApp based on whether the current time falls inside hoursJSON.
// Called after saving the "operational" AI prompt section so the change takes effect
// without waiting for the scheduler's next minute tick.
func (cont *SettingContImpl) syncBotStateFromOperationalHours(schema, hoursJSON string) {
	if hoursJSON == "" {
		return
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
	manualMode := "true"
	botEnabled := "false"
	if withinHours {
		manualMode = "false"
		botEnabled = "true"
	}

	for _, platform := range []string{"Telegram", "WhatsApp"} {
		cont.Db.Exec(fmt.Sprintf(
			`INSERT INTO %s.setting (group_name, sub_group_name, name, value)
			 VALUES ('integration', $1, 'manual-mode', $2)
			 ON CONFLICT (sub_group_name, name) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			schema,
		), platform, manualMode)

		cont.Db.Exec(fmt.Sprintf(
			`INSERT INTO %s.setting (group_name, sub_group_name, name, value)
			 VALUES ('integration', $1, 'bot-enabled', $2)
			 ON CONFLICT (sub_group_name, name) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			schema,
		), platform, botEnabled)
	}
}

// UploadKnowledgeBase godoc
// @Summary      Upload a PDF to the knowledge base
// @Tags         Settings
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string  true  "Client ID"
// @Param        file       formData  file    true  "PDF file (max 10 MB)"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      400        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/knowledge-base [post]
func (cont *SettingContImpl) UploadKnowledgeBase(ctx *gin.Context) {
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

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		helpers.WriteToResponseBody(ctx, 400, helpers.ApiResponse{Success: false, Code: 400, Data: "No file uploaded"})
		return
	}

	if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".pdf") {
		helpers.WriteToResponseBody(ctx, 400, helpers.ApiResponse{Success: false, Code: 400, Data: "Only PDF files are allowed"})
		return
	}

	const maxSize = 10 * 1024 * 1024
	if fileHeader.Size > maxSize {
		helpers.WriteToResponseBody(ctx, 400, helpers.ApiResponse{Success: false, Code: 400, Data: "File must be at most 10 MB"})
		return
	}

	extractedText := extractPDFWithTimeout(fileHeader)

	uploaded, uploadErr := cont.FileServ.UploadFiles([]*multipart.FileHeader{fileHeader})
	if uploadErr != nil || len(uploaded) == 0 {
		helpers.WriteToResponseBody(ctx, 500, helpers.ApiResponse{Success: false, Code: 500, Data: "Failed to upload file"})
		return
	}

	kb := domains.KnowledgeBase{
		FileName:      fileHeader.Filename,
		FileURL:       uploaded[0].FileURL,
		ExtractedText: extractedText,
	}
	created, dbErr := cont.KnowledgeBaseRepo.Create(cont.Db, schema, kb)
	if dbErr != nil {
		helpers.WriteToResponseBody(ctx, 500, helpers.ApiResponse{Success: false, Code: 500, Data: "Failed to save knowledge base entry"})
		return
	}

	_ = helpers.WriteToResponseBody(ctx, 200, helpers.ApiResponse{Success: true, Code: 200, Data: created})
}

// ListKnowledgeBase godoc
// @Summary      List knowledge base files
// @Tags         Settings
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true  "Client ID"
// @Success      200        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/knowledge-base [get]
func (cont *SettingContImpl) ListKnowledgeBase(ctx *gin.Context) {
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

	items, err := cont.KnowledgeBaseRepo.FindAll(cont.Db, schema)
	if err != nil {
		helpers.WriteToResponseBody(ctx, 500, helpers.ApiResponse{Success: false, Code: 500, Data: "Failed to load knowledge base"})
		return
	}

	_ = helpers.WriteToResponseBody(ctx, 200, helpers.ApiResponse{Success: true, Code: 200, Data: items})
}

// DeleteKnowledgeBase godoc
// @Summary      Delete a knowledge base file
// @Tags         Settings
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true  "Client ID"
// @Param        kb_id      path  string  true  "Knowledge Base entry ID"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      400        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/knowledge-base/{kb_id} [delete]
func (cont *SettingContImpl) DeleteKnowledgeBase(ctx *gin.Context) {
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

	idStr := ctx.Param("kb_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.WriteToResponseBody(ctx, 400, helpers.ApiResponse{Success: false, Code: 400, Data: "Invalid ID"})
		return
	}

	kb, err := cont.KnowledgeBaseRepo.FindByID(cont.Db, schema, id)
	if err != nil {
		helpers.WriteToResponseBody(ctx, 404, helpers.ApiResponse{Success: false, Code: 404, Data: "Knowledge base entry not found"})
		return
	}

	if err := cont.KnowledgeBaseRepo.Delete(cont.Db, schema, id); err != nil {
		helpers.WriteToResponseBody(ctx, 500, helpers.ApiResponse{Success: false, Code: 500, Data: "Failed to delete"})
		return
	}

	_ = cont.FileServ.DeleteFile(kb.FileURL)

	_ = helpers.WriteToResponseBody(ctx, 200, helpers.ApiResponse{Success: true, Code: 200, Data: nil})
}

func extractPDFWithTimeout(fh *multipart.FileHeader) string {
	type result struct {
		text string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		f, err := fh.Open()
		if err != nil {
			ch <- result{"", err}
			return
		}
		defer f.Close()
		text, err := extractPDFText(f)
		ch <- result{text, err}
	}()
	select {
	case r := <-ch:
		if r.err != nil {
			return ""
		}
		return r.text
	case <-time.After(15 * time.Second):
		return ""
	}
}

func extractPDFText(r io.Reader) (string, error) {
	tmp, err := os.CreateTemp("", "kb-*.pdf")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return "", err
	}
	tmp.Close()

	f, pdfReader, err := pdf.Open(tmpName)
	if err != nil {
		return "", err
	}
	defer f.Close()

	plainText, err := pdfReader.GetPlainText()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(plainText); err != nil {
		return "", err
	}

	text := strings.TrimSpace(buf.String())
	const maxLen = 20000
	if len(text) > maxLen {
		text = text[:maxLen]
	}
	return text, nil
}
