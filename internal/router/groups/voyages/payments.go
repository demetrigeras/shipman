package voyages

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"shipman/internal/coinsub"
	"shipman/internal/db"
)

type PaymentHandler struct {
	paymentRepo *db.PaymentRepository
	voyageRepo  *db.VoyageRepository
	userRepo    *db.UserRepository
	coinsub     *coinsub.Client
	appURL      string
}

func NewPaymentHandler(coinsubClient *coinsub.Client, appURL string) *PaymentHandler {
	return &PaymentHandler{
		paymentRepo: db.NewPaymentRepository(),
		voyageRepo:  db.NewVoyageRepository(),
		userRepo:    db.NewUserRepository(),
		coinsub:     coinsubClient,
		appURL:      appURL,
	}
}

func (h *PaymentHandler) AddRoutes(r *gin.RouterGroup) {
	r.GET("/:id/payments", h.handleList)
	r.POST("/:id/payments", h.handleCreate)
	r.POST("/:id/payments/:paymentId/checkout", h.handleCheckout)
	r.POST("/:id/payments/:paymentId/mark-paid", h.handleMarkPaid)
	r.POST("/:id/payments/:paymentId/transfer", h.handleTransfer)
	r.DELETE("/:id/payments/:paymentId", h.handleDelete)
}

func (h *PaymentHandler) AddUserRoutes(r *gin.RouterGroup) {
	// User routes for future use
}

func (h *PaymentHandler) AddAdminRoutes(r *gin.RouterGroup) {
	r.POST("/coinsub/register-webhook", h.handleRegisterWebhook)
	r.GET("/coinsub/status", h.handleCoinsubStatus)
}

func (h *PaymentHandler) AddPublicRoutes(r *gin.RouterGroup) {
	r.POST("/webhooks/coinsub", h.handleWebhook)
}

// canAccessVoyage returns true if the user is owner, counterparty, or
// broker on the voyage. Primary check is against the user_id columns set
// by handleJoinVoyage. The email fallback is kept for legacy data where
// somebody was set as counterparty by email before user-id columns existed.
func (h *PaymentHandler) canAccessVoyage(ctx context.Context, v db.Voyage, userID uuid.UUID) bool {
	if v.OwnerUserID != nil && *v.OwnerUserID == userID {
		return true
	}
	if v.CounterpartyUserID != nil && *v.CounterpartyUserID == userID {
		return true
	}
	if v.BrokerUserID != nil && *v.BrokerUserID == userID {
		return true
	}
	if v.CounterpartyEmail != nil {
		u, err := h.userRepo.Retrieve(ctx, userID)
		if err == nil && u.Email == *v.CounterpartyEmail {
			return true
		}
	}
	return false
}

type CreatePaymentRequest struct {
	PaymentType string  `json:"payment_type" binding:"required"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount" binding:"required"`
	Currency    string  `json:"currency"`
	Recurring   bool    `json:"recurring"`
	Interval    string  `json:"interval"`
	Frequency   string  `json:"frequency"`
}

func (h *PaymentHandler) handleList(c *gin.Context) {
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}
	payments, err := h.paymentRepo.ListByVoyage(c.Request.Context(), voyageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list payments"})
		return
	}
	if payments == nil {
		payments = []db.VoyagePayment{}
	}
	c.JSON(http.StatusOK, payments)
}

func (h *PaymentHandler) handleCreate(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}

	v, err := h.voyageRepo.Retrieve(c.Request.Context(), voyageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "voyage not found"})
		return
	}
	if !h.canAccessVoyage(c.Request.Context(), v, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	payment := &db.VoyagePayment{
		VoyageID:    voyageID,
		CreatedBy:   userID,
		PaymentType: req.PaymentType,
		Amount:      req.Amount,
		Currency:    currency,
		Status:      "draft",
	}
	name := req.Name
	if name != "" {
		payment.Description = &name
	} else if req.Description != "" {
		payment.Description = &req.Description
	}
	// Pre-fill recipient with the voyage counterparty so invoice cards can show who it's sent to
	if v.CounterpartyEmail != nil && *v.CounterpartyEmail != "" {
		payment.RecipientEmail = v.CounterpartyEmail
	}

	if err := h.paymentRepo.Create(c.Request.Context(), payment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create payment"})
		return
	}

	// Auto-create Coinsub purchase session so the invoice is immediately payable
	if h.coinsub.Enabled() {
		fixtureTitle := "Unnamed Fixture"
		if v.VoyageNumber != nil {
			fixtureTitle = *v.VoyageNumber
		} else if v.VesselName != nil {
			fixtureTitle = *v.VesselName
		}
		sessionName := fixtureTitle + " — " + payment.PaymentType
		details := payment.PaymentType + " payment"
		if payment.Description != nil {
			sessionName = *payment.Description
			details = *payment.Description
		}
		metadata := map[string]string{
			"payment_id": payment.ID.String(),
			"voyage_id":  voyageID.String(),
		}
		if v.CargoType != nil {
			metadata["cargo_type"] = *v.CargoType
		}
		if v.VesselName != nil {
			metadata["vessel_name"] = *v.VesselName
		}

		sessReq := coinsub.CreateSessionRequest{
			Name:           sessionName,
			Details:        details,
			Amount:         payment.Amount,
			Currency:       currency,
			SuccessURL:     h.appURL + "/voyages/" + voyageID.String() + "?tab=payments&status=success",
			CancelURL:      h.appURL + "/voyages/" + voyageID.String() + "?tab=payments&status=cancelled",
			ExpiresInHours: 72,
			Metadata:       metadata,
		}
		if req.Recurring {
			sessReq.Recurring = true
			sessReq.Interval = req.Interval
			sessReq.Frequency = req.Frequency
			if sessReq.Frequency == "" {
				sessReq.Frequency = "Every"
			}
			sessReq.Duration = "Until Cancelled"
		}

		result, err := h.coinsub.CreatePurchaseSession(sessReq)
		if err != nil {
			log.Printf("coinsub auto-session failed (payment %s): %v", payment.ID, err)
		} else {
			_ = h.paymentRepo.UpdateCoinsubSession(c.Request.Context(), payment.ID, result.Data.PurchaseSessionID, result.Data.URL)
			sessionID := result.Data.PurchaseSessionID
			checkoutURL := result.Data.URL
			payment.CoinsubSessionID = &sessionID
			payment.CoinsubCheckoutURL = &checkoutURL
			payment.Status = "pending"
		}
	}

	c.JSON(http.StatusCreated, payment)
}

func (h *PaymentHandler) handleCheckout(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}
	paymentID, err := uuid.Parse(c.Param("paymentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment ID"})
		return
	}

	v, err := h.voyageRepo.Retrieve(c.Request.Context(), voyageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "voyage not found"})
		return
	}
	if !h.canAccessVoyage(c.Request.Context(), v, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if !h.coinsub.Enabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Coinsub payments not configured. Set COINSUB_API_KEY and COINSUB_MERCHANT_ID."})
		return
	}

	payment, err := h.paymentRepo.Retrieve(c.Request.Context(), paymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}
	if payment.VoyageID != voyageID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment does not belong to this voyage"})
		return
	}

	// Parse optional recurring params from the request body
	var body struct {
		Recurring bool   `json:"recurring"`
		Interval  string `json:"interval"`
		Frequency string `json:"frequency"`
		Duration  string `json:"duration"`
	}
	_ = c.ShouldBindJSON(&body)

	fixtureTitle := "Unnamed Fixture"
	if v.VoyageNumber != nil {
		fixtureTitle = *v.VoyageNumber
	} else if v.VesselName != nil {
		fixtureTitle = *v.VesselName
	}

	sessionName := fixtureTitle + " — " + payment.PaymentType
	details := payment.PaymentType + " payment"
	if payment.Description != nil {
		sessionName = *payment.Description
		details = *payment.Description
	}

	// Build metadata from voyage context
	metadata := map[string]string{
		"payment_id": payment.ID.String(),
		"voyage_id":  voyageID.String(),
	}
	if v.CargoType != nil {
		metadata["cargo_type"] = *v.CargoType
	}
	if v.VesselName != nil {
		metadata["vessel_name"] = *v.VesselName
	}

	req := coinsub.CreateSessionRequest{
		Name:           sessionName,
		Details:        details,
		Amount:         payment.Amount,
		Currency:       payment.Currency,
		Recurring:      body.Recurring,
		SuccessURL:     h.appURL + "/voyages/" + voyageID.String() + "?tab=payments&status=success",
		CancelURL:      h.appURL + "/voyages/" + voyageID.String() + "?tab=payments&status=cancelled",
		ExpiresInHours: 24,
		Metadata:       metadata,
	}
	if body.Recurring {
		req.Interval = body.Interval
		req.Frequency = body.Frequency
		if req.Frequency == "" {
			req.Frequency = "Every"
		}
		req.Duration = body.Duration
		if req.Duration == "" {
			req.Duration = "Until Cancelled"
		}
	}

	result, err := h.coinsub.CreatePurchaseSession(req)
	if err != nil {
		log.Printf("coinsub session create failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create Coinsub checkout session", "details": err.Error()})
		return
	}

	if err := h.paymentRepo.UpdateCoinsubSession(c.Request.Context(), paymentID, result.Data.PurchaseSessionID, result.Data.URL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"checkout_url": result.Data.URL,
		"session_id":   result.Data.PurchaseSessionID,
	})
}

func (h *PaymentHandler) handleTransfer(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}
	paymentID, err := uuid.Parse(c.Param("paymentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment ID"})
		return
	}

	v, err := h.voyageRepo.Retrieve(c.Request.Context(), voyageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "voyage not found"})
		return
	}
	if !h.canAccessVoyage(c.Request.Context(), v, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if !h.coinsub.Enabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Coinsub payments not configured"})
		return
	}

	payment, err := h.paymentRepo.Retrieve(c.Request.Context(), paymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}

	var reqBody struct {
		ToAddress string `json:"to_address" binding:"required"`
		ChainID   int    `json:"chain_id"`
		Token     string `json:"token"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to_address is required (recipient email or wallet)"})
		return
	}
	chainID := reqBody.ChainID
	if chainID == 0 {
		chainID = 137
	}
	token := reqBody.Token
	if token == "" {
		token = payment.Currency
		if token == "USD" {
			token = "USDC"
		}
	}

	result, err := h.coinsub.CreateTransfer(coinsub.TransferRequest{
		ToAddress: reqBody.ToAddress,
		Amount:    payment.Amount,
		ChainID:   chainID,
		Token:     token,
	})
	if err != nil {
		log.Printf("coinsub transfer failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "transfer failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": result.Data.Message,
		"fee":     result.Data.Fee,
	})
}

func (h *PaymentHandler) handleMarkPaid(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}
	paymentID, err := uuid.Parse(c.Param("paymentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment ID"})
		return
	}

	v, err := h.voyageRepo.Retrieve(c.Request.Context(), voyageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "voyage not found"})
		return
	}
	if !h.canAccessVoyage(c.Request.Context(), v, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := h.paymentRepo.MarkPaid(c.Request.Context(), paymentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark payment as paid"})
		return
	}

	payment, err := h.paymentRepo.Retrieve(c.Request.Context(), paymentID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "marked as paid"})
		return
	}
	c.JSON(http.StatusOK, payment)
}

func (h *PaymentHandler) handleDelete(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}
	paymentID, err := uuid.Parse(c.Param("paymentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment ID"})
		return
	}

	v, err := h.voyageRepo.Retrieve(c.Request.Context(), voyageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "voyage not found"})
		return
	}
	if !h.canAccessVoyage(c.Request.Context(), v, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := h.paymentRepo.Delete(c.Request.Context(), paymentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete payment"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *PaymentHandler) handleWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	timestamp := c.GetHeader("X-Webhook-Timestamp")
	signature := c.GetHeader("X-Webhook-Signature")
	if !h.coinsub.VerifyWebhook(timestamp, signature, body) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
		return
	}

	var payload coinsub.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook payload"})
		return
	}

	log.Printf("coinsub webhook received: type=%s status=%s", payload.Type, payload.Status)

	switch payload.Type {
	case "payment":
		// Look up our payment record by metadata.payment_id first, then fall back to session lookup
		paymentUUID := ""
		if payload.Metadata != nil {
			paymentUUID = payload.Metadata["payment_id"]
		}
		payerEmail := payload.User.Email
		if paymentUUID != "" {
			if pid, err := uuid.Parse(paymentUUID); err == nil {
				if err := h.paymentRepo.MarkCompletedByID(c.Request.Context(), pid, payload.PaymentID, payload.TransactionDetails.TransactionHash, payerEmail); err != nil {
					log.Printf("MarkCompletedByID failed for payment %s: %v", paymentUUID, err)
				}
			}
		} else if payload.OriginID != "" {
			// Fall back to session-ID lookup
			h.paymentRepo.MarkCompleted(c.Request.Context(), payload.OriginID, payload.PaymentID, payload.TransactionDetails.TransactionHash)
		}

	case "failed_payment":
		paymentUUID := ""
		if payload.Metadata != nil {
			paymentUUID = payload.Metadata["payment_id"]
		}
		if paymentUUID != "" {
			if pid, err := uuid.Parse(paymentUUID); err == nil {
				h.paymentRepo.MarkFailedByID(c.Request.Context(), pid)
			}
		} else if payload.OriginID != "" {
			h.paymentRepo.MarkFailed(c.Request.Context(), payload.OriginID)
		}

	case "transfer":
		log.Printf("coinsub transfer webhook: hash=%s status=%s to=%s", payload.Hash, payload.Status, payload.ToAddress)
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// ── Admin: webhook registration ─────────────────────────────────────

func (h *PaymentHandler) handleRegisterWebhook(c *gin.Context) {
	if !h.coinsub.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Coinsub not configured"})
		return
	}

	var body struct {
		WebhookURL string `json:"webhook_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "webhook_url is required"})
		return
	}

	result, err := h.coinsub.CreateWebhook("", body.WebhookURL)
	if err != nil {
		log.Printf("coinsub webhook registration failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to register webhook with Coinsub", "details": err.Error()})
		return
	}

	h.coinsub.SetWebhookSecret(result.Data.SigningSecret)

	c.JSON(http.StatusOK, gin.H{
		"message":        result.Data.Message,
		"webhook_id":     result.Data.WebhookID,
		"signing_secret": result.Data.SigningSecret,
		"status":         result.Data.Status,
		"note":           "IMPORTANT: Save the signing_secret in your COINSUB_WEBHOOK_SECRET env var for persistence across restarts.",
	})
}

func (h *PaymentHandler) handleCoinsubStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"enabled":     h.coinsub.Enabled(),
		"merchant_id": h.coinsub.MerchantID(),
		"webhook_url": h.appURL + "/api/v1/webhooks/coinsub",
	})
}

func splitName(full string) [2]string {
	parts := [2]string{full, ""}
	for i, ch := range full {
		if ch == ' ' {
			parts[0] = full[:i]
			parts[1] = full[i+1:]
			break
		}
	}
	return parts
}
