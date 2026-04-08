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

const WhatsAppAPIURL = "https://graph.facebook.com/v20.0"

type WhatsAppClient struct {
	PhoneNumberID string
	AccessToken   string
	Client        *http.Client
}

func NewWhatsAppClient(phoneNumberID, accessToken string) *WhatsAppClient {
	return &WhatsAppClient{
		PhoneNumberID: phoneNumberID,
		AccessToken:   accessToken,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *WhatsAppClient) SendMessage(to, text string) error {
	if c.PhoneNumberID == "" || c.AccessToken == "" {
		return fmt.Errorf("whatsapp credentials not configured")
	}

	reqBody := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text":              map[string]string{"body": text},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s/messages", WhatsAppAPIURL, c.PhoneNumberID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("[WhatsApp] sendMessage to=%s status=%d body=%s", to, resp.StatusCode, string(body))

	if resp.StatusCode != 200 {
		return fmt.Errorf("whatsapp API error: status %d", resp.StatusCode)
	}

	return nil
}
