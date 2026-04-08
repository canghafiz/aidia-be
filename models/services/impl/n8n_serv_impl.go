package impl

import (
	"backend/models/domains"
	"backend/models/services"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type N8NServImpl struct {
	N8NURL       string
	APIKey       string
	HTTPClient   *http.Client
}

// N8NRequest represents request to n8n webhook
type N8NRequest struct {
	Schema   string                  `json:"schema"`
	GuestID  string                  `json:"guest_id"`
	ChatID   string                  `json:"chat_id"`
	Message  string                  `json:"message"`
	Prompt   string                  `json:"prompt,omitempty"`
	History  []N8NMessageHistory     `json:"history,omitempty"`
	Metadata map[string]interface{}  `json:"metadata,omitempty"`
}

// N8NMessageHistory represents message history for context
type N8NMessageHistory struct {
	Role      string    `json:"role"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func NewN8NServImpl(webhookURL string) *N8NServImpl {
	return &N8NServImpl{
		N8NURL: webhookURL,
		APIKey: os.Getenv("N8N_API_KEY"),
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				TLSHandshakeTimeout:   30 * time.Second,
				ResponseHeaderTimeout: 60 * time.Second,
			},
		},
	}
}

func NewTelegramN8NServImpl() *N8NServImpl {
	return NewN8NServImpl(os.Getenv("N8N_TELEGRAM_WEBHOOK_URL"))
}

func NewWhatsAppN8NServImpl() *N8NServImpl {
	return NewN8NServImpl(os.Getenv("N8N_WHATSAPP_WEBHOOK_URL"))
}

// ProcessMessage forwards message to n8n for AI processing
func (s *N8NServImpl) ProcessMessage(schema, guestID, chatID, message, prompt string, history []domains.GuestMessage) (*services.N8NResponse, error) {
	if s.N8NURL == "" {
		return nil, fmt.Errorf("n8n webhook URL not configured")
	}

	// Use default prompt if not provided
	if prompt == "" {
		prompt = "Anda adalah asisten AI untuk restoran ini. Tugas Anda:\n1. Bantu customer lihat menu/produk\n2. Bantu customer buat pesanan\n3. Jawab pertanyaan seputar restoran\n4. Selalu konfirmasi sebelum membuat pesanan\n\nBalas dengan bahasa yang ramah dan natural."
	}

	// Build request
	reqBody := N8NRequest{
		Schema:  schema,
		GuestID: guestID,
		ChatID:  chatID,
		Message: message,
		Prompt:  prompt,
	}

	// Add message history (last 20 messages for context)
	maxHistory := 20
	if len(history) < maxHistory {
		maxHistory = len(history)
	}

	for i := 0; i < maxHistory; i++ {
		msg := history[i]
		reqBody.History = append(reqBody.History, N8NMessageHistory{
			Role:      msg.Role,
			Message:   msg.Message,
			Timestamp: msg.CreatedAt,
		})
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", s.N8NURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	
	// Add API key if configured
	if s.APIKey != "" {
		req.Header.Set("X-API-Key", s.APIKey)
	}

	// Send request
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to n8n: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("n8n returned status: %d", resp.StatusCode)
	}

	// Parse response
	var n8nResp services.N8NResponse
	if err := json.NewDecoder(resp.Body).Decode(&n8nResp); err != nil {
		return nil, fmt.Errorf("failed to decode n8n response: %w", err)
	}

	return &n8nResp, nil
}

var _ services.N8NServ = (*N8NServImpl)(nil)
