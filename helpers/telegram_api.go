package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const TelegramAPIURL = "https://api.telegram.org/bot"

type TelegramClient struct {
	Token  string
	Client *http.Client
}

func NewTelegramClient(token string) *TelegramClient {
	return &TelegramClient{
		Token: token,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type SendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type SendMessageResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	Result      struct {
		MessageID int `json:"message_id"`
		Chat      struct {
			ID int `json:"id"`
		} `json:"chat"`
	} `json:"result"`
}

type GetMeResponse struct {
	OK     bool `json:"ok"`
	Result Bot  `json:"result"`
}

type Bot struct {
	ID                      int    `json:"id"`
	IsBot                   bool   `json:"is_bot"`
	FirstName               string `json:"first_name"`
	Username                string `json:"username"`
	CanJoinGroups           bool   `json:"can_join_groups"`
	CanReadAllGroupMessages bool   `json:"can_read_all_group_messages"`
	SupportsInlineQueries   bool   `json:"supports_inline_queries"`
}

type SetWebhookRequest struct {
	URL string `json:"url"`
}

type SetWebhookResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

type ReplyKeyboardMarkup struct {
	Keyboard        [][]KeyboardButton `json:"keyboard"`
	ResizeKeyboard  bool               `json:"resize_keyboard"`
	OneTimeKeyboard bool               `json:"one_time_keyboard"`
}

type KeyboardButton struct {
	Text            string `json:"text"`
	RequestContact  bool   `json:"request_contact,omitempty"`
	RequestLocation bool   `json:"request_location,omitempty"`
}

func (c *TelegramClient) doPost(endpoint string, reqBody map[string]interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s%s/%s", TelegramAPIURL, c.Token, endpoint)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("[Telegram] %s status: %d, body: %s", endpoint, resp.StatusCode, string(body))
	return body, nil
}

func (c *TelegramClient) SendMessage(chatID, text string) (*SendMessageResponse, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("telegram token is empty")
	}

	body, err := c.doPost("sendMessage", map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	})
	if err != nil {
		return nil, err
	}

	var telegramResp SendMessageResponse
	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !telegramResp.OK {
		return nil, fmt.Errorf("telegram API error: %s", telegramResp.Description)
	}

	return &telegramResp, nil
}

func (c *TelegramClient) SendMessageWithKeyboard(chatID, text string, keyboard *ReplyKeyboardMarkup) error {
	if c.Token == "" {
		return fmt.Errorf("telegram token is empty")
	}

	reqBody := map[string]interface{}{
		"chat_id":      chatID,
		"text":         text,
		"reply_markup": keyboard,
	}

	jsonCheck, _ := json.Marshal(reqBody)
	log.Printf("[Telegram] SendMessageWithKeyboard RAW JSON: %s", string(jsonCheck))

	body, err := c.doPost("sendMessage", reqBody)
	if err != nil {
		return err
	}

	var telegramResp SendMessageResponse
	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !telegramResp.OK {
		return fmt.Errorf("telegram API error: %s", telegramResp.Description)
	}

	log.Printf("[Telegram] ✅ SendMessageWithKeyboard success to %s", chatID)
	return nil
}

func (c *TelegramClient) SendMessageWithInlineKeyboard(chatID, text string, inlineKeyboard map[string]interface{}) error {
	if c.Token == "" {
		return fmt.Errorf("telegram token is empty")
	}

	body, err := c.doPost("sendMessage", map[string]interface{}{
		"chat_id":      chatID,
		"text":         text,
		"reply_markup": inlineKeyboard,
	})
	if err != nil {
		return err
	}

	var telegramResp SendMessageResponse
	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !telegramResp.OK {
		return fmt.Errorf("telegram API error: %s", telegramResp.Description)
	}

	log.Printf("[Telegram] ✅ SendMessageWithInlineKeyboard success to %s", chatID)
	return nil
}

func (c *TelegramClient) AnswerCallbackQuery(callbackQueryID, text string) error {
	if c.Token == "" {
		return fmt.Errorf("telegram token is empty")
	}

	body, err := c.doPost("answerCallbackQuery", map[string]interface{}{
		"callback_query_id": callbackQueryID,
		"text":              text,
		"show_alert":        false,
	})
	if err != nil {
		return err
	}

	var telegramResp SendMessageResponse
	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *TelegramClient) SetWebhook(webhookURL string) (*SetWebhookResponse, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("telegram token is empty")
	}

	body, err := c.doPost("setWebhook", map[string]interface{}{
		"url": webhookURL,
	})
	if err != nil {
		return nil, err
	}

	var telegramResp SetWebhookResponse
	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !telegramResp.OK {
		return nil, fmt.Errorf("telegram API error: %s", telegramResp.Description)
	}

	return &telegramResp, nil
}

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var telegramResp GetMeResponse
	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !telegramResp.OK {
		return nil, fmt.Errorf("telegram API error: invalid token")
	}

	return &telegramResp, nil
}
