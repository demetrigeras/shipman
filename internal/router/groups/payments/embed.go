// Package payments hosts cross-cutting payment endpoints that aren't tied
// to a specific resource (deal/voyage). Right now it's just the RocketRamp
// embed-code minter, but anything generic (refunds, payout history, etc.)
// would live here.
package payments

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"shipman/internal/rocketramp"
)

// Handler is the embed-code endpoint group.
type Handler struct {
	rocket *rocketramp.Client
}

// NewHandler constructs the embed-code handler.
func NewHandler(rocket *rocketramp.Client) *Handler {
	return &Handler{rocket: rocket}
}

// AddRoutes wires the routes under an already-authenticated group.
func (h *Handler) AddRoutes(r *gin.RouterGroup) {
	r.POST("/embed-code", h.handleCreateEmbedCode)
	r.GET("/embed-config", h.handleEmbedConfig)
}

type createEmbedCodeRequest struct {
	RecipientEmail string `json:"recipient_email" binding:"required,email"`
	Memo           string `json:"memo"`
	// Amount in USD. Optional; when > 0 the RR /send screen pre-fills the
	// amount field too (useful for payment-request flows).
	Amount *float64 `json:"amount"`
}

type createEmbedCodeResponse struct {
	EmbedCode    string `json:"embed_code"`
	EmbedBaseURL string `json:"embed_base_url"`
	// EmbedURL is the full URL the FE should open in a popup. We build it
	// server-side so the URL-shape decision (`?s=<code>` vs `/embed/<code>`)
	// stays one place we can tweak without redeploying the FE.
	EmbedURL string `json:"embed_url"`
	TestMode bool   `json:"test_mode"`
}

// POST /api/v1/payments/embed-code
//
//	{ "recipient_email": "alice@example.com", "memo": "Voyage #42 commission" }
//
// Returns a single-use UUID the FE embeds in <iframe src="{embed_base_url}/{embed_code}">.
func (h *Handler) handleCreateEmbedCode(c *gin.Context) {
	var req createEmbedCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	req.RecipientEmail = strings.TrimSpace(req.RecipientEmail)
	req.Memo = strings.TrimSpace(req.Memo)

	code, err := h.rocket.CreateEmbedCode(c.Request.Context(), req.RecipientEmail, req.Memo, req.Amount)
	if err != nil {
		if errors.Is(err, rocketramp.ErrNotConfigured) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "RocketRamp not configured",
				"details": "Set ROCKETRAMP_MERCHANT_ID and ROCKETRAMP_API_KEY environment variables on the backend.",
			})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, createEmbedCodeResponse{
		EmbedCode:    code,
		EmbedBaseURL: h.rocket.EmbedBaseURL(),
		EmbedURL:     h.rocket.EmbedURL(code),
		TestMode:     h.rocket.TestMode(),
	})
}

// GET /api/v1/payments/embed-config — lets the FE know whether payments are
// available (so it can hide the Pay buttons if RocketRamp isn't configured).
func (h *Handler) handleEmbedConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"enabled":        h.rocket.Enabled(),
		"test_mode":      h.rocket.TestMode(),
		"embed_base_url": h.rocket.EmbedBaseURL(),
	})
}
