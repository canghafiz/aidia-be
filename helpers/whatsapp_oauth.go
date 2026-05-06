package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

// WAPhoneNumber represents a phone number from the Meta Graph API
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

// ExchangeCodeForToken exchanges an OAuth code from Embedded Signup for an access token
func ExchangeCodeForToken(code string) (string, error) {
	appID := os.Getenv("META_APP_ID")
	appSecret := os.Getenv("META_APP_SECRET")

	if appID == "" || appSecret == "" {
		return "", fmt.Errorf("META_APP_ID or META_APP_SECRET is not configured")
	}

	// For Embedded Signup (JS SDK), redirect_uri must not be included
	params := url.Values{}
	params.Set("client_id", appID)
	params.Set("client_secret", appSecret)
	params.Set("code", code)

	apiURL := fmt.Sprintf("%s/oauth/access_token?%s", WhatsAppAPIURL, params.Encode())
	resp, err := metaHTTPClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result metaTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("Meta API error: %s (code %d)", result.Error.Message, result.Error.Code)
	}

	if result.AccessToken == "" {
		return "", fmt.Errorf("access token returned empty from Meta")
	}

	return result.AccessToken, nil
}

// ExtendToken exchanges a short-lived token for a long-lived token (~60 days)
func ExtendToken(shortToken string) (string, error) {
	appID := os.Getenv("META_APP_ID")
	appSecret := os.Getenv("META_APP_SECRET")

	if appID == "" || appSecret == "" {
		return "", fmt.Errorf("META_APP_ID or META_APP_SECRET is not configured")
	}

	params := url.Values{}
	params.Set("grant_type", "fb_exchange_token")
	params.Set("client_id", appID)
	params.Set("client_secret", appSecret)
	params.Set("fb_exchange_token", shortToken)

	apiURL := fmt.Sprintf("%s/oauth/access_token?%s", WhatsAppAPIURL, params.Encode())
	resp, err := metaHTTPClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to extend token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result metaTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("Meta API error: %s (code %d)", result.Error.Message, result.Error.Code)
	}

	if result.AccessToken == "" {
		// If Meta does not return a new token, use the existing token
		return shortToken, nil
	}

	return result.AccessToken, nil
}

// GetWABAPhoneNumbers retrieves the list of phone numbers for a given WABA
func GetWABAPhoneNumbers(wabaID, accessToken string) ([]WAPhoneNumber, error) {
	apiURL := fmt.Sprintf("%s/%s/phone_numbers?access_token=%s", WhatsAppAPIURL, wabaID, accessToken)

	resp, err := metaHTTPClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch phone numbers: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result metaPhoneNumbersResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse phone numbers: %w", err)
	}

	return result.Data, nil
}

// ExchangeCodeForTokenWithURI exchanges an OAuth code using an explicit redirect URI
// Used for OAuth Redirect Flow (not Embedded Signup)
func ExchangeCodeForTokenWithURI(code, redirectURI string) (string, error) {
	appID := os.Getenv("META_APP_ID")
	appSecret := os.Getenv("META_APP_SECRET")

	if appID == "" || appSecret == "" {
		return "", fmt.Errorf("META_APP_ID or META_APP_SECRET is not configured")
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
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result metaTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("Meta API error: %s (code %d)", result.Error.Message, result.Error.Code)
	}

	if result.AccessToken == "" {
		return "", fmt.Errorf("access token returned empty from Meta")
	}

	return result.AccessToken, nil
}

// WABAInfo represents a WhatsApp Business Account
type WABAInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type metaWABAResponse struct {
	Data []WABAInfo `json:"data"`
}

type metaBusinessInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type metaBusinessResponse struct {
	Data []metaBusinessInfo `json:"data"`
}

// GetUserWABAs retrieves the list of WhatsApp Business Accounts accessible by the user.
// Layered strategy:
// (1) /me/whatsapp_business_accounts — WABAs directly owned by the user
// (2) /me/businesses → owned + client WABAs — WABAs under Business Manager
// (3) /me/businesses?fields=client_whatsapp_business_accounts — WABAs shared to this business
func GetUserWABAs(accessToken string) ([]WABAInfo, error) {
	seen := map[string]bool{}
	var all []WABAInfo

	addUnique := func(wabas []WABAInfo) {
		for _, w := range wabas {
			if !seen[w.ID] {
				seen[w.ID] = true
				all = append(all, w)
			}
		}
	}

	// Log granted permissions for diagnosis
	permURL := fmt.Sprintf("%s/me/permissions?access_token=%s", WhatsAppAPIURL, accessToken)
	if permResp, err := metaHTTPClient.Get(permURL); err == nil {
		defer permResp.Body.Close()
		permBody, _ := io.ReadAll(permResp.Body)
		log.Printf("[WA OAuth] /me/permissions: %s", string(permBody))
	}

	// 1. Directly on the user
	directURL := fmt.Sprintf("%s/me/whatsapp_business_accounts?fields=id,name&access_token=%s", WhatsAppAPIURL, accessToken)
	direct, directRaw, _ := fetchWABAsFromURLDebug(directURL)
	log.Printf("[WA OAuth] /me/whatsapp_business_accounts → %d results raw=%s", len(direct), directRaw)
	addUnique(direct)
	if len(all) > 0 {
		return all, nil
	}

	// 2. Via Business Manager — fetch the list of businesses first, then fetch WABAs per business
	bizURL := fmt.Sprintf("%s/me/businesses?fields=id,name&access_token=%s", WhatsAppAPIURL, accessToken)
	resp, err := metaHTTPClient.Get(bizURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch businesses: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	log.Printf("[WA OAuth] /me/businesses raw response: %s", string(body))

	var bizResult metaBusinessResponse
	if err := json.Unmarshal(body, &bizResult); err != nil {
		return nil, fmt.Errorf("failed to parse businesses: %w", err)
	}

	log.Printf("[WA OAuth] businesses found: %d", len(bizResult.Data))
	for _, biz := range bizResult.Data {
		log.Printf("[WA OAuth] checking business id=%s name=%s", biz.ID, biz.Name)

		// Owned WABAs
		ownedURL := fmt.Sprintf("%s/%s/whatsapp_business_accounts?fields=id,name&access_token=%s", WhatsAppAPIURL, biz.ID, accessToken)
		owned, ownedRaw, ownedErr := fetchWABAsFromURLDebug(ownedURL)
		log.Printf("[WA OAuth]   owned WABAs: %d err=%v raw=%s", len(owned), ownedErr, ownedRaw)
		addUnique(owned)

		// Client WABAs (shared to this business)
		clientURL := fmt.Sprintf("%s/%s/client_whatsapp_business_accounts?fields=id,name&access_token=%s", WhatsAppAPIURL, biz.ID, accessToken)
		client, clientRaw, clientErr := fetchWABAsFromURLDebug(clientURL)
		log.Printf("[WA OAuth]   client WABAs: %d err=%v raw=%s", len(client), clientErr, clientRaw)
		addUnique(client)
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("no accessible WhatsApp Business account found")
	}
	return all, nil
}

func fetchWABAsFromURL(apiURL string) ([]WABAInfo, error) {
	wabas, _, err := fetchWABAsFromURLDebug(apiURL)
	return wabas, err
}

func fetchWABAsFromURLDebug(apiURL string) ([]WABAInfo, string, error) {
	resp, err := metaHTTPClient.Get(apiURL)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	raw := string(body)
	var result metaWABAResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, raw, err
	}
	return result.Data, raw, nil
}

// SubscribeAppToWABA registers our app to the webhook events of a tenant's WABA
func SubscribeAppToWABA(wabaID, accessToken string) error {
	apiURL := fmt.Sprintf("%s/%s/subscribed_apps", WhatsAppAPIURL, wabaID)

	req, err := http.NewRequest("POST", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := metaHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[WA OAuth] subscribed_apps response status=%d body=%s", resp.StatusCode, string(body))
	if resp.StatusCode != 200 {
		return fmt.Errorf("subscribe failed status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}
