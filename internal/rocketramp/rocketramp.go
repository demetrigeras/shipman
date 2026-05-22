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
	// Embed host roots. We keep these as roots (no `/embed` suffix) so we
	// can construct either form of the popup URL:
	//   - Root + "/?s=<code>"  (preferred: feeds default_route.go's hidden
	//     `prefilledSessionId` input directly, skipping ShowEmbedPrefill)
	//   - Root + "/embed/<code>" (legacy: relies on ShowEmbedPrefill to
	//     redirect to /?s=<code>)
	prodEmbedRoot = "https://app.myrocketramp.com"
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

// EmbedBaseURL returns the legacy `<root>/embed` path. Kept for backwards
// compatibility with the FE which previously embedded `${embed_base_url}/${code}`.
func (c *Client) EmbedBaseURL() string {
	return c.embedRoot() + "/embed"
}

// embedRoot returns the wallet host root (no `/embed` suffix).
func (c *Client) embedRoot() string {

	return prodEmbedRoot
}

// EmbedURL builds the full popup URL we want the browser to open for a
// given embed_code. We use the `?s=<code>` form (NOT `/embed/<code>`) because
// the diagnosis from the RR team showed that:
//   - `/embed/<code>` goes through ShowEmbedPrefill, which then redirects to
//     `/?s=<code>`. The redirect chain occasionally drops the session
//     identifier, leaving the sign-in form's hidden `prefilledSessionId`
//     input empty → VerifyOTP then doesn't hydrate prefill data → user
//     lands on /home instead of /send.
//   - `/?s=<code>` hits default_route.go directly with the query param the
//     template needs, so the hidden input is always populated and OTP
//     verify always finds the prefill session.
func (c *Client) EmbedURL(embedCode string) string {
	return c.embedRoot() + "/?s=" + embedCode
}

func (c *Client) apiBase() string {
	if c.testMode {
		return sandboxAPIBase
	}
	return prodAPIBase
}

type prefillRequest struct {
	Email  string   `json:"email"`
	Memo   string   `json:"memo,omitempty"`
	Amount *float64 `json:"amount,omitempty"`
}

type prefillResponse struct {
	EmbedCode string `json:"embed_code"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

// CreateEmbedCode mints a fresh single-use embed_code for the given
// recipient email. The returned UUID can be passed to the FE which opens
// the wallet at <embedRoot>/?s=<embedCode>.
//
// `amount` is optional. When non-nil and > 0 the RR /send screen will
// pre-fill (and lock) the amount field after OTP succeeds.
func (c *Client) CreateEmbedCode(ctx context.Context, recipientEmail, memo string, amount *float64) (string, error) {
	if !c.Enabled() {
		return "", ErrNotConfigured
	}

	payload := prefillRequest{Email: recipientEmail, Memo: memo}
	// RR rejects amount<=0, so only attach when meaningful.
	if amount != nil && *amount > 0 {
		payload.Amount = amount
	}
	body, err := json.Marshal(payload)
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
