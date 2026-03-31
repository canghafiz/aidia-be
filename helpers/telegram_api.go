package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const TelegramAPIURL = "https://api.telegram.org/bot"

// TelegramClient handles Telegram Bot API calls
type TelegramClient struct {
	Token  string
	Client *http.Client
}

// NewTelegramClient creates a new Telegram client
func NewTelegramClient(token string) *TelegramClient {
	return &TelegramClient{
		Token: token,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendMessageRequest represents Telegram sendMessage request
type SendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// SendMessageResponse represents Telegram sendMessage response
type SendMessageResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		MessageID int `json:"message_id"`
		Chat      struct {
			ID int `json:"id"`
		} `json:"chat"`
	} `json:"result"`
}

// SendMessage sends a text message via Telegram Bot API
func (c *TelegramClient) SendMessage(chatID, text string) (*SendMessageResponse, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("telegram token is empty")
	}

	reqBody := SendMessageRequest{
		ChatID: chatID,
		Text:   text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s%s/sendMessage", TelegramAPIURL, c.Token)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var telegramResp SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !telegramResp.OK {
		return nil, fmt.Errorf("telegram API error: response not ok")
	}

	return &telegramResp, nil
}

// SendMessageWithKeyboard sends a message with custom keyboard (for contact share button)
func (c *TelegramClient) SendMessageWithKeyboard(chatID, text string, keyboard map[string]interface{}) error {
	if c.Token == "" {
		return fmt.Errorf("telegram token is empty")
	}

	reqBody := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
		"reply_markup": keyboard,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s%s/sendMessage", TelegramAPIURL, c.Token)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var telegramResp SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !telegramResp.OK {
		return fmt.Errorf("telegram API error: response not ok")
	}

	return nil
}

// SetWebhookRequest represents Telegram setWebhook request
type SetWebhookRequest struct {
	URL string `json:"url"`
}

// SetWebhookResponse represents Telegram setWebhook response
type SetWebhookResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

// SetWebhook registers webhook URL to Telegram
func (c *TelegramClient) SetWebhook(webhookURL string) (*SetWebhookResponse, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("telegram token is empty")
	}

	reqBody := SetWebhookRequest{
		URL: webhookURL,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s%s/setWebhook", TelegramAPIURL, c.Token)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var telegramResp SetWebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !telegramResp.OK {
		return nil, fmt.Errorf("telegram API error: %s", telegramResp.Description)
	}

	return &telegramResp, nil
}

// GetMeResponse represents Telegram getMe response
type GetMeResponse struct {
	OK       bool `json:"ok"`
	Result   Bot  `json:"result"`
}

// Bot represents Telegram bot info
type Bot struct {
	ID        int    `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
	CanJoinGroups bool `json:"can_join_groups"`
	CanReadAllGroupMessages bool `json:"can_read_all_group_messages"`
	SupportsInlineQueries bool `json:"supports_inline_queries"`
}

// GetMe gets bot info
func (c *TelegramClient) GetMe() (*GetMeResponse, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("telegram token is empty")
	}

	url := fmt.Sprintf("%s%s/getMe", TelegramAPIURL, c.Token)

	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var telegramResp GetMeResponse
	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !telegramResp.OK {
		return nil, fmt.Errorf("telegram API error: invalid token")
	}

	return &telegramResp, nil
}
