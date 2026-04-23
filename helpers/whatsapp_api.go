package helpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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

type waContactsRequest struct {
	Blocking   string   `json:"blocking"`
	Contacts   []string `json:"contacts"`
	ForceCheck bool     `json:"force_check"`
}

type waContactsResponse struct {
	Contacts []struct {
		Input  string `json:"input"`
		Status string `json:"status"`
		WaID   string `json:"wa_id,omitempty"`
	} `json:"contacts"`
}

type metaGraphAPIErrorResponse struct {
	Error *struct {
		Message      string `json:"message"`
		Type         string `json:"type"`
		Code         int    `json:"code"`
		ErrorSubcode int    `json:"error_subcode"`
		ErrorUserTitle string `json:"error_user_title"`
		ErrorUserMsg   string `json:"error_user_msg"`
	} `json:"error"`
}

type WhatsAppValidationUnsupportedError struct {
	Message string
}

func (e *WhatsAppValidationUnsupportedError) Error() string {
	if e == nil || e.Message == "" {
		return "whatsapp number validation is unsupported for this configuration"
	}
	return e.Message
}

func IsWhatsAppValidationUnsupported(err error) bool {
	var target *WhatsAppValidationUnsupportedError
	return errors.As(err, &target)
}

type WhatsAppRecipientNotRegisteredError struct {
	Message string
}

func (e *WhatsAppRecipientNotRegisteredError) Error() string {
	if e == nil || e.Message == "" {
		return "recipient number is not registered on WhatsApp"
	}
	return e.Message
}

func IsWhatsAppRecipientNotRegistered(err error) bool {
	var target *WhatsAppRecipientNotRegisteredError
	return errors.As(err, &target)
}

type WhatsAppBusinessNotRegisteredError struct {
	Message string
}

func (e *WhatsAppBusinessNotRegisteredError) Error() string {
	if e == nil || e.Message == "" {
		return "whatsapp business phone number is not registered with Cloud API"
	}
	return e.Message
}

func IsWhatsAppBusinessNotRegistered(err error) bool {
	var target *WhatsAppBusinessNotRegisteredError
	return errors.As(err, &target)
}

type WhatsAppRegistrationBlockedError struct {
	Message string
}

func (e *WhatsAppRegistrationBlockedError) Error() string {
	if e == nil || e.Message == "" {
		return "whatsapp business phone number cannot be activated yet"
	}
	return e.Message
}

func IsWhatsAppRegistrationBlocked(err error) bool {
	var target *WhatsAppRegistrationBlockedError
	return errors.As(err, &target)
}

// CheckPhoneExists checks whether a phone number is registered on WhatsApp.
// countryCode should be digits only (e.g. "62"), phoneNumber digits only (e.g. "81234567890").
func (c *WhatsAppClient) CheckPhoneExists(countryCode, phoneNumber string) (bool, error) {
	if c.PhoneNumberID == "" || c.AccessToken == "" {
		return false, fmt.Errorf("whatsapp credentials not configured")
	}

	reqBody := waContactsRequest{
		Blocking:   "wait",
		Contacts:   []string{"+" + countryCode + phoneNumber},
		ForceCheck: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return false, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s/contacts", WhatsAppAPIURL, c.PhoneNumberID)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("[WhatsApp] checkContacts status=%d body=%s", resp.StatusCode, string(body))

	if resp.StatusCode != 200 {
		var graphErr metaGraphAPIErrorResponse
		if err := json.Unmarshal(body, &graphErr); err == nil && graphErr.Error != nil {
			if graphErr.Error.Code == 100 && graphErr.Error.ErrorSubcode == 33 {
				return false, &WhatsAppValidationUnsupportedError{
					Message: fmt.Sprintf("whatsapp number validation is unsupported for phone_number_id %s: %s", c.PhoneNumberID, graphErr.Error.Message),
				}
			}
			if strings.Contains(strings.ToLower(graphErr.Error.Message), "unsupported post request") {
				return false, &WhatsAppValidationUnsupportedError{
					Message: graphErr.Error.Message,
				}
			}
		}
		return false, fmt.Errorf("whatsapp API error: status %d", resp.StatusCode)
	}

	var contactsResp waContactsResponse
	if err := json.Unmarshal(body, &contactsResp); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(contactsResp.Contacts) == 0 {
		return false, nil
	}

	return contactsResp.Contacts[0].Status == "valid", nil
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
		var graphErr metaGraphAPIErrorResponse
		if err := json.Unmarshal(body, &graphErr); err == nil && graphErr.Error != nil {
			if graphErr.Error.Code == 133010 {
				return &WhatsAppBusinessNotRegisteredError{
					Message: "whatsapp business phone number is not registered with Cloud API",
				}
			}
			if graphErr.Error.Message != "" {
				return fmt.Errorf("whatsapp API error: %s", graphErr.Error.Message)
			}
		}
		return fmt.Errorf("whatsapp API error: status %d", resp.StatusCode)
	}

	return nil
}

func (c *WhatsAppClient) SendTemplateMessage(to, templateName, languageCode string, bodyParams []string) error {
	if c.PhoneNumberID == "" || c.AccessToken == "" {
		return fmt.Errorf("whatsapp credentials not configured")
	}
	templateName = strings.TrimSpace(templateName)
	languageCode = strings.TrimSpace(languageCode)
	if templateName == "" {
		return fmt.Errorf("template name is required")
	}
	if languageCode == "" {
		languageCode = "en_US"
	}

	templatePayload := map[string]interface{}{
		"name": templateName,
		"language": map[string]string{
			"code": languageCode,
		},
	}

	if len(bodyParams) > 0 {
		parameters := make([]map[string]string, 0, len(bodyParams))
		for _, param := range bodyParams {
			parameters = append(parameters, map[string]string{
				"type": "text",
				"text": param,
			})
		}
		templatePayload["components"] = []map[string]interface{}{
			{
				"type":       "body",
				"parameters": parameters,
			},
		}
	}

	reqBody := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "template",
		"template":          templatePayload,
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

	log.Printf("[WhatsApp] sendTemplate to=%s template=%s status=%d body=%s", to, templateName, resp.StatusCode, string(body))

	if resp.StatusCode != 200 {
		var graphErr metaGraphAPIErrorResponse
		if err := json.Unmarshal(body, &graphErr); err == nil && graphErr.Error != nil {
			if graphErr.Error.Message != "" {
				return fmt.Errorf("whatsapp template error: %s", graphErr.Error.Message)
			}
		}
		return fmt.Errorf("whatsapp template error: status %d", resp.StatusCode)
	}

	return nil
}

func (c *WhatsAppClient) RegisterPhoneNumber(pin string) error {
	if c.PhoneNumberID == "" || c.AccessToken == "" {
		return fmt.Errorf("whatsapp credentials not configured")
	}
	pin = strings.TrimSpace(pin)
	if pin == "" {
		return fmt.Errorf("whatsapp registration pin is not configured")
	}

	reqBody := map[string]interface{}{
		"messaging_product": "whatsapp",
		"pin":               pin,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s/register", WhatsAppAPIURL, c.PhoneNumberID)
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

	log.Printf("[WhatsApp] registerPhoneNumber status=%d body=%s", resp.StatusCode, string(body))

	if resp.StatusCode != 200 {
		var graphErr metaGraphAPIErrorResponse
		if err := json.Unmarshal(body, &graphErr); err == nil && graphErr.Error != nil {
			if graphErr.Error.Code == 100 && graphErr.Error.ErrorSubcode == 2388001 {
				msg := graphErr.Error.ErrorUserMsg
				if strings.TrimSpace(msg) == "" {
					msg = graphErr.Error.Message
				}
				return &WhatsAppRegistrationBlockedError{Message: msg}
			}
			if graphErr.Error.Message != "" {
				return fmt.Errorf("whatsapp register error: %s", graphErr.Error.Message)
			}
		}
		return fmt.Errorf("whatsapp register error: status %d", resp.StatusCode)
	}

	return nil
}
