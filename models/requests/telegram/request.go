package telegram

type UpdateAIPromptRequest struct {
	Prompt string `json:"prompt" validate:"required,max=2000"`
}

type UpdateBotTokenRequest struct {
	BotToken string `json:"bot_token" validate:"required"`
}

type RequestPhoneRequest struct {
	ChatID string `json:"chat_id" validate:"required"`
}

type AIPromptResponse struct {
	Prompt string `json:"prompt"`
}
