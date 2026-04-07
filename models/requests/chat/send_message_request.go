package chat

// SendMessageRequest is the request body for sending a manual reply to a guest.
type SendMessageRequest struct {
	Message string `json:"message" binding:"required" example:"Halo, ada yang bisa kami bantu?"`
}
