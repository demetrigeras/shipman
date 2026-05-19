package coinsub

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	apiKey        string
	merchantID    string
	webhookSecret string
	baseURL       string
}

func NewClient(apiKey, merchantID, webhookSecret string) *Client {
	return &Client{
		apiKey:        apiKey,
		merchantID:    merchantID,
		webhookSecret: webhookSecret,
		baseURL:       "https://api.coinsub.io",
	}
}

func (c *Client) SetTestMode() {
	c.baseURL = "https://test-api.coinsub.io"
}

func (c *Client) Enabled() bool {
	return c.apiKey != "" && c.merchantID != ""
}

// ── Submerchant Accounts ─────────────────────────────────────────────────

type CreateSubmerchantRequest struct {
	DefaultDepositAddress string                    `json:"default_deposit_address,omitempty"`
	BusinessProfile       *SubmerchantBizProfile    `json:"business_profile,omitempty"`
	Individual            *SubmerchantIndividual    `json:"individual,omitempty"`
	Settings              *SubmerchantSettings      `json:"settings,omitempty"`
}

type SubmerchantBizProfile struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Email       string `json:"email,omitempty"`
	CompanyName string `json:"company_name,omitempty"`
}

type SubmerchantIndividual struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Email     string `json:"email,omitempty"`
}

type SubmerchantSettings struct {
	Branding *SubmerchantBranding `json:"branding,omitempty"`
}

type SubmerchantBranding struct {
	BrandColor string `json:"brand_color,omitempty"`
}

type CreateSubmerchantResponse struct {
	Data struct {
		MerchantID            string `json:"merchant_id"`
		SubmerchantID         string `json:"submerchant_id"`
		EmailVerificationLink string `json:"email_verification_link"`
	} `json:"data"`
	Status int `json:"status"`
}

func (c *Client) CreateSubmerchant(req CreateSubmerchantRequest) (*CreateSubmerchantResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequest("POST", c.baseURL+"/v1/merchants/create", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("API-Key", c.apiKey)
	httpReq.Header.Set("Merchant-ID", c.merchantID)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("coinsub create submerchant error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result CreateSubmerchantResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse coinsub response: %w", err)
	}
	return &result, nil
}

// ── Webhooks ─────────────────────────────────────────────────────────

type CreateWebhookRequest struct {
	URL string `json:"url"`
}

type CreateWebhookResponse struct {
	Data struct {
		Message       string `json:"message"`
		WebhookID     int    `json:"webhook_id"`
		Status        string `json:"status"`
		SigningSecret string `json:"signing_secret"`
	} `json:"data"`
	Status int `json:"status"`
}

// CreateWebhook registers a webhook URL for the given merchant.
// Use merchantID="" to register for the platform merchant itself.
func (c *Client) CreateWebhook(merchantID, webhookURL string) (*CreateWebhookResponse, error) {
	if merchantID == "" {
		merchantID = c.merchantID
	}
	body, _ := json.Marshal(CreateWebhookRequest{URL: webhookURL})
	url := fmt.Sprintf("%s/v1/merchants/%s/webhooks", c.baseURL, merchantID)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("API-Key", c.apiKey)
	httpReq.Header.Set("Merchant-ID", c.merchantID)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("coinsub create webhook error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result CreateWebhookResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse coinsub response: %w", err)
	}
	return &result, nil
}

// SetWebhookSecret updates the client's webhook secret after registration.
func (c *Client) SetWebhookSecret(secret string) {
	c.webhookSecret = secret
}

func (c *Client) MerchantID() string {
	return c.merchantID
}

// ── Purchase Sessions ─────────────────────────────────────────────────

type CreateSessionRequest struct {
	Name           string            `json:"name"`
	Details        string            `json:"details"`
	Amount         float64           `json:"amount"`
	Currency       string            `json:"currency"`
	Recurring      bool              `json:"recurring"`
	Interval       string            `json:"interval,omitempty"`       // Day, Week, Month, Year (required if recurring)
	Frequency      string            `json:"frequency,omitempty"`      // Every, Every Other, Every Third, etc.
	Duration       string            `json:"Duration,omitempty"`       // "Until Cancelled" or a number
	SuccessURL     string            `json:"success_url,omitempty"`
	CancelURL      string            `json:"cancel_url,omitempty"`
	ExpiresInHours int               `json:"expires_in_hours,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type CreateSessionResponse struct {
	Data struct {
		PurchaseSessionID string  `json:"purchase_session_id"`
		URL               string  `json:"url"`
		Amount            float64 `json:"amount"`
		Currency          string  `json:"currency"`
		Status            string  `json:"status"`
	} `json:"data"`
	Status int `json:"status"`
}

func (c *Client) CreatePurchaseSession(req CreateSessionRequest) (*CreateSessionResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequest("POST", c.baseURL+"/v1/purchase/session/start", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("API-Key", c.apiKey)
	httpReq.Header.Set("Merchant-ID", c.merchantID)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("coinsub API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result CreateSessionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse coinsub response: %w", err)
	}
	return &result, nil
}

// ── Cancel Agreement ──────────────────────────────────────────────────

func (c *Client) CancelAgreement(agreementID string) error {
	url := fmt.Sprintf("%s/v1/agreements/cancel/%s", c.baseURL, agreementID)
	httpReq, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("API-Key", c.apiKey)
	httpReq.Header.Set("Merchant-ID", c.merchantID)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("coinsub cancel agreement error (status %d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

type TransferRequest struct {
	ToAddress string  `json:"to_address"`
	Amount    float64 `json:"amount"`
	ChainID   int     `json:"chainId"`
	Token     string  `json:"token"`
}

type TransferResponse struct {
	Data struct {
		Fee     float64 `json:"fee"`
		Message string  `json:"message"`
		Status  string  `json:"status"`
	} `json:"data"`
	Status int `json:"status"`
}

func (c *Client) CreateTransfer(req TransferRequest) (*TransferResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequest("POST", c.baseURL+"/v1/merchants/transfer/request", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("API-Key", c.apiKey)
	httpReq.Header.Set("Merchant-ID", c.merchantID)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("coinsub transfer error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result TransferResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse coinsub response: %w", err)
	}
	return &result, nil
}

// WebhookPayload handles both payment/failed_payment and transfer webhook types.
type WebhookPayload struct {
	Type               string            `json:"type"`
	MerchantID         string            `json:"merchant_id"`
	Status             string            `json:"status"`

	// Payment fields
	OriginID           string            `json:"origin_id,omitempty"`
	Origin             string            `json:"origin,omitempty"`
	Name               string            `json:"name,omitempty"`
	Currency           string            `json:"currency,omitempty"`
	Amount             float64           `json:"amount,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	PaymentDate        string            `json:"payment_date,omitempty"`
	PaymentID          string            `json:"payment_id,omitempty"`
	AgreementID        string            `json:"agreement_id,omitempty"`
	TransactionDetails struct {
		TransactionID   int    `json:"transaction_id"`
		TransactionHash string `json:"transaction_hash"`
		ChainID         int    `json:"chain_id"`
	} `json:"transaction_details,omitempty"`
	User struct {
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		Email        string `json:"email"`
		SubscriberID string `json:"subscriber_id"`
	} `json:"user,omitempty"`

	// Transfer fields
	AmountInUSD       string `json:"amount_in_usd,omitempty"`
	Hash              string `json:"hash,omitempty"`
	TransferID        string `json:"transfer_id,omitempty"`
	WalletID          string `json:"wallet_id,omitempty"`
	Network           string `json:"network,omitempty"`
	FromAddress       string `json:"from_address,omitempty"`
	ToAddress         string `json:"to_address,omitempty"`
	StatusConfirmedAt string `json:"status_confirmed_at,omitempty"`
}

func (c *Client) VerifyWebhook(timestamp, signature string, body []byte) bool {
	if c.webhookSecret == "" {
		return true
	}
	payload := timestamp + "." + string(body)
	mac := hmac.New(sha256.New, []byte(c.webhookSecret))
	mac.Write([]byte(payload))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
