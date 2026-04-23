package helpers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

const (
	hitPayBaseURL        = "https://api.hit-pay.com/v1"
	hitPaySandboxBaseURL = "https://api.sandbox.hit-pay.com/v1"
)

type HitPayPaymentRequest struct {
	Amount          string
	Currency        string
	Email           string
	Name            string
	ReferenceNumber string
	RedirectURL     string
	WebhookURL      string
	Purpose         string
}

type HitPayPaymentResponse struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Status string `json:"status"`
}

// HitPayCreatePayment creates a payment request via HitPay API and returns
// the payment ID and redirect URL.
func HitPayCreatePayment(apiKey string, req HitPayPaymentRequest, sandbox bool) (*HitPayPaymentResponse, error) {
	baseURL := hitPayBaseURL
	if sandbox {
		baseURL = hitPaySandboxBaseURL
	}

	formData := url.Values{}
	formData.Set("amount", req.Amount)
	formData.Set("currency", req.Currency)
	formData.Set("email", req.Email)
	formData.Set("name", req.Name)
	formData.Set("reference_number", req.ReferenceNumber)
	formData.Set("redirect_url", req.RedirectURL)
	formData.Set("webhook", req.WebhookURL)
	formData.Set("purpose", req.Purpose)

	httpReq, err := http.NewRequest(http.MethodPost, baseURL+"/payment-requests", strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("hitpay: failed to build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("X-BUSINESS-API-KEY", apiKey)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("hitpay: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("hitpay: failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hitpay: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result HitPayPaymentResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("hitpay: failed to parse response: %w", err)
	}

	return &result, nil
}

// HitPayVerifyWebhook verifies the HMAC-SHA256 signature included in the
// HitPay webhook POST body. formValues must contain all fields from the body,
// including "hmac". Returns true when the signature is valid.
func HitPayVerifyWebhook(formValues map[string]string, webhookSalt string) bool {
	receivedHMAC, ok := formValues["hmac"]
	if !ok || receivedHMAC == "" {
		return false
	}

	keys := make([]string, 0, len(formValues))
	for k := range formValues {
		if k != "hmac" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+formValues[k])
	}

	mac := hmac.New(sha256.New, []byte(webhookSalt))
	mac.Write([]byte(strings.Join(parts, "&")))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(receivedHMAC))
}
