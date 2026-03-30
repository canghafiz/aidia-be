package telegram

type UpdateAIPromptRequest struct {
	Prompt string `json:"prompt" validate:"required,max=2000"`
}

type AIPromptResponse struct {
	Prompt string `json:"prompt"`
}
