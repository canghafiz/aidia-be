package chat

// SendMessageRequest is the request body for sending a manual reply to a guest.
type SendMessageRequest struct {
	Message string `json:"message" binding:"required" example:"Halo, ada yang bisa kami bantu?"`
}

type SendTemplateMessageRequest struct {
	TemplateName string   `json:"template_name" binding:"required" example:"hello_world"`
	LanguageCode string   `json:"language_code" example:"en_US"`
	BodyParams   []string `json:"body_params"`
}
