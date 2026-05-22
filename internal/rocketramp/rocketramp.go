// Package rocketramp wraps the Vantack Prefill API for the RocketRamp
// embed wallet. The merchant credentials must stay server-side; the only
// thing we ever return to the browser is a single-use embed_code UUID.
//
// PRODUCTION ONLY. Sandbox / test URLs have been intentionally removed —
// the popup, the Vantack API, and the wallet host all point at the live
// RocketRamp environment regardless of any `ROCKETRAMP_TEST_MODE` env var
// or YAML config. The TestMode flag/method is kept around purely so the
// existing call sites compile; it has no effect on URLs anymore.
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
	apiBaseURL   = "https://api.vantack.com"
	embedRootURL = "https://app.myrocketramp.com"
)

// Client talks to the Vantack Prefill API.
type Client struct {
	merchantID string
	apiKey     string
	http       *http.Client
}

// NewClient returns a configured Vantack client. If merchantID or apiKey are
// empty the client is "disabled" and CreateEmbedCode returns ErrNotConfigured.
// The testMode parameter is accepted for backwards compatibility but is
// ignored — every call goes to production.
func NewClient(merchantID, apiKey string, _testMode bool) *Client {
	return &Client{
		merchantID: merchantID,
		apiKey:     apiKey,
		http:       &http.Client{Timeout: 10 * time.Second},
	}
}

// ErrNotConfigured is returned when no credentials are configured.
var ErrNotConfigured = errors.New("rocketramp credentials are not configured")

// Enabled reports whether the client has credentials set.
func (c *Client) Enabled() bool {
	return c.merchantID != "" && c.apiKey != ""
}

// TestMode always returns false — sandbox has been removed.
func (c *Client) TestMode() bool { return false }

// EmbedBaseURL returns the legacy `<root>/embed` path. Kept for backwards
// compatibility with FE callers that previously built URLs as
// `${embed_base_url}/${code}`.
func (c *Client) EmbedBaseURL() string {
	return embedRootURL + "/embed"
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
	return embedRootURL + "/?s=" + embedCode
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
// the wallet at <embedRootURL>/?s=<embedCode>.
//
// `amount` is optional. When non-nil and > 0 the RR /send screen will
// pre-fill (and lock) the amount field after OTP succeeds.
func (c *Client) CreateEmbedCode(ctx context.Context, recipientEmail, memo string, amount *float64) (string, error) {
	if !c.Enabled() {
		return "", ErrNotConfigured
	}

	payload := prefillRequest{Email: recipientEmail, Memo: memo}
	if amount != nil && *amount > 0 {
		payload.Amount = amount
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal prefill body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBaseURL+"/v1/merchants/embed/prefill", bytes.NewReader(body))
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
