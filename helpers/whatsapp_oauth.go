package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// WAPhoneNumber representasi nomor telepon dari Meta Graph API
type WAPhoneNumber struct {
	ID                 string `json:"id"`
	DisplayPhoneNumber string `json:"display_phone_number"`
	VerifiedName       string `json:"verified_name"`
}

type metaTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

type metaPhoneNumbersResponse struct {
	Data []WAPhoneNumber `json:"data"`
}

var metaHTTPClient = &http.Client{Timeout: 30 * time.Second}

// ExchangeCodeForToken menukar OAuth code dari Embedded Signup menjadi access token
func ExchangeCodeForToken(code string) (string, error) {
	appID := os.Getenv("META_APP_ID")
	appSecret := os.Getenv("META_APP_SECRET")
	redirectURI := os.Getenv("META_REDIRECT_URI")

	if appID == "" || appSecret == "" {
		return "", fmt.Errorf("META_APP_ID atau META_APP_SECRET belum dikonfigurasi")
	}

	params := url.Values{}
	params.Set("client_id", appID)
	params.Set("client_secret", appSecret)
	params.Set("code", code)
	if redirectURI != "" {
		params.Set("redirect_uri", redirectURI)
	}

	apiURL := fmt.Sprintf("%s/oauth/access_token?%s", WhatsAppAPIURL, params.Encode())
	resp, err := metaHTTPClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("gagal request token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result metaTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("gagal parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("Meta API error: %s (code %d)", result.Error.Message, result.Error.Code)
	}

	if result.AccessToken == "" {
		return "", fmt.Errorf("access token kosong dari Meta")
	}

	return result.AccessToken, nil
}

// ExtendToken menukar short-lived token menjadi long-lived token (~60 hari)
func ExtendToken(shortToken string) (string, error) {
	appID := os.Getenv("META_APP_ID")
	appSecret := os.Getenv("META_APP_SECRET")

	if appID == "" || appSecret == "" {
		return "", fmt.Errorf("META_APP_ID atau META_APP_SECRET belum dikonfigurasi")
	}

	params := url.Values{}
	params.Set("grant_type", "fb_exchange_token")
	params.Set("client_id", appID)
	params.Set("client_secret", appSecret)
	params.Set("fb_exchange_token", shortToken)

	apiURL := fmt.Sprintf("%s/oauth/access_token?%s", WhatsAppAPIURL, params.Encode())
	resp, err := metaHTTPClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("gagal extend token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result metaTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("gagal parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("Meta API error: %s (code %d)", result.Error.Message, result.Error.Code)
	}

	if result.AccessToken == "" {
		// Jika Meta tidak mengembalikan token baru, gunakan token lama
		return shortToken, nil
	}

	return result.AccessToken, nil
}

// GetWABAPhoneNumbers mengambil daftar nomor telepon dari sebuah WABA
func GetWABAPhoneNumbers(wabaID, accessToken string) ([]WAPhoneNumber, error) {
	apiURL := fmt.Sprintf("%s/%s/phone_numbers?access_token=%s", WhatsAppAPIURL, wabaID, accessToken)

	resp, err := metaHTTPClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("gagal fetch phone numbers: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result metaPhoneNumbersResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gagal parse phone numbers: %w", err)
	}

	return result.Data, nil
}

// SubscribeAppToWABA mendaftarkan app kita ke webhook events WABA milik tenant
func SubscribeAppToWABA(wabaID, accessToken string) error {
	apiURL := fmt.Sprintf("%s/%s/subscribed_apps", WhatsAppAPIURL, wabaID)

	req, err := http.NewRequest("POST", apiURL, nil)
	if err != nil {
		return fmt.Errorf("gagal buat request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := metaHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("gagal subscribe: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("subscribe gagal status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}
