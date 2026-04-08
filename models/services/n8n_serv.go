package services

import (
	"backend/models/domains"
)

// N8NResponse represents response from n8n webhook
type N8NResponse struct {
	Reply       string                 `json:"reply"`
	UsageTokens int64                  `json:"usage_tokens,omitempty"`
	Intent      string                 `json:"intent,omitempty"`
	Entities    map[string]interface{} `json:"entities,omitempty"`
	Action      *N8NAction             `json:"action,omitempty"`
}

// N8NAction represents action to execute
type N8NAction struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

type N8NServ interface {
	ProcessMessage(schema, guestID, chatID, message, prompt string, history []domains.GuestMessage) (*N8NResponse, error)
}
