// Package rocketramp wraps the Vantack Prefill API for the RocketRamp
// embed wallet. The merchant credentials must stay server-side; the only
// thing we ever return to the browser is a single-use embed_code UUID.
package rocketramp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	prodAPIBase    = "https://api.vantack.com"
	sandboxAPIBase = "https://test-api.vantack.com"
	prodEmbedBase    = "https://app.myrocketramp.com/embed"
	sandboxEmbedBase = "https://test.myrocketramp.com/embed"
)

// Client talks to the Vantack Prefill API.
type Client struct {
	merchantID string
	apiKey     string
	testMode   bool
	http       *http.Client
}

// NewClient returns a configured Vantack client. If merchantID or apiKey are
// empty the client is "disabled" and CreateEmbedCode returns ErrNotConfigured.
func NewClient(merchantID, apiKey string, testMode bool) *Client {
	return &Client{
		merchantID: merchantID,
		apiKey:     apiKey,
		testMode:   testMode,
		http:       &http.Client{Timeout: 10 * time.Second},
	}
}

// ErrNotConfigured is returned when no credentials are configured.
var ErrNotConfigured = errors.New("rocketramp credentials are not configured")

// Enabled reports whether the client has credentials set.
func (c *Client) Enabled() bool {
	return c.merchantID != "" && c.apiKey != ""
}

// TestMode reports whether the client is in sandbox mode.
func (c *Client) TestMode() bool { return c.testMode }

// EmbedBaseURL returns the host the FE should load the iframe from for the
// current environment (so the FE doesn't have to guess).
func (c *Client) EmbedBaseURL() string {
	if c.testMode {
		return sandboxEmbedBase
	}
	return prodEmbedBase
}

func (c *Client) apiBase() string {
	if c.testMode {
		return sandboxAPIBase
	}
	return prodAPIBase
}

type prefillRequest struct {
	Email string `json:"email"`
	Memo  string `json:"memo,omitempty"`
}

type prefillResponse struct {
	EmbedCode string `json:"embed_code"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

// CreateEmbedCode mints a fresh single-use embed_code for the given
// recipient email. The returned UUID can be passed to the FE which embeds
// the wallet iframe at <EmbedBaseURL>/<embedCode>.
func (c *Client) CreateEmbedCode(ctx context.Context, recipientEmail, memo string) (string, error) {
	if !c.Enabled() {
		return "", ErrNotConfigured
	}

	body, err := json.Marshal(prefillRequest{Email: recipientEmail, Memo: memo})
	if err != nil {
		return "", fmt.Errorf("marshal prefill body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBase()+"/v1/merchants/embed/prefill", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build prefill request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Merchant-ID", c.merchantID)
	req.Header.Set("API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("call vantack prefill: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var parsed prefillResponse
		_ = json.Unmarshal(respBody, &parsed)
		if parsed.Error != "" {
			return "", fmt.Errorf("vantack prefill %d: %s", resp.StatusCode, parsed.Error)
		}
		return "", fmt.Errorf("vantack prefill %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed prefillResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("parse vantack response: %w", err)
	}
	if parsed.EmbedCode == "" {
		return "", fmt.Errorf("vantack returned empty embed_code")
	}
	return parsed.EmbedCode, nil
}
