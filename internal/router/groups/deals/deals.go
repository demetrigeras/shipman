package deals

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"shipman/internal/db"
	"shipman/internal/email"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	dealRepo    *db.DealRepository
	negRepo     *db.NegotiationRepository
	detailsRepo *db.DealDetailsRepository
	emailSvc    *email.Service
	appURL      string
}

func NewHandler(emailSvc *email.Service, appURL string) *Handler {
	return &Handler{
		dealRepo:    db.NewDealRepository(),
		negRepo:     db.NewNegotiationRepository(),
		detailsRepo: db.NewDealDetailsRepository(),
		emailSvc:    emailSvc,
		appURL:      appURL,
	}
}

// AddPublicRoutes registers routes that don't need authentication (invite preview).
func (h *Handler) AddPublicRoutes(r *gin.RouterGroup) {
	r.GET("/invite/:token", h.handleGetInvitePreview)
}

// AddRoutes registers routes that require authentication.
func (h *Handler) AddRoutes(r *gin.RouterGroup) {
	r.POST("", h.handleCreate)
	r.GET("", h.handleList)
	r.GET("/:id", h.handleGet)
	r.POST("/:id/invite", h.handleCreateInvite)
	r.POST("/join", h.handleJoinDeal)
	r.PATCH("/:id/document", h.handleAttachDocument)
	r.PUT("/:id/vessel", h.handleUpsertVesselDetails)
	r.PUT("/:id/cargo", h.handleUpsertCargoDetails)
	r.POST("/:id/negotiations", h.handleCreateNegotiation)
	r.GET("/:id/negotiations", h.handleListNegotiations)
	r.POST("/:id/negotiations/:negotiationId/proposals", h.handleCreateProposal)
	r.GET("/:id/negotiations/:negotiationId", h.handleGetNegotiation)
	r.PATCH("/:id/negotiations/:negotiationId/proposals/:proposalId", h.handleUpdateProposalStatus)
}

type CreateDealRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description *string `json:"description"`
	DocumentID  *string `json:"document_id"`
}

func (h *Handler) handleCreate(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req CreateDealRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deal := &db.Deal{
		Title:       req.Title,
		Description: req.Description,
		Status:      "active",
		CreatedBy:   userID,
	}

	if req.DocumentID != nil {
		docID, err := uuid.Parse(*req.DocumentID)
		if err == nil {
			deal.DocumentID = &docID
		}
	}

	if err := h.dealRepo.Create(c.Request.Context(), deal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create deal"})
		return
	}

	userRole := c.GetString("userRole")
	now := time.Now()
	participant := &db.DealParticipant{
		DealID:   deal.ID,
		UserID:   &userID,
		Role:     userRole,
		JoinedAt: &now,
	}
	h.dealRepo.AddParticipant(c.Request.Context(), participant)

	c.JSON(http.StatusCreated, deal)
}

func (h *Handler) handleList(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	deals, err := h.dealRepo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list deals"})
		return
	}

	if deals == nil {
		deals = []db.Deal{}
	}

	c.JSON(http.StatusOK, gin.H{"data": deals})
}

func (h *Handler) handleGet(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	dealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deal ID"})
		return
	}

	isParticipant, err := h.dealRepo.IsParticipant(c.Request.Context(), dealID, userID)
	if err != nil || !isParticipant {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	deal, err := h.dealRepo.Retrieve(c.Request.Context(), dealID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "deal not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve deal"})
		return
	}

	participants, _ := h.dealRepo.GetParticipants(c.Request.Context(), dealID)
	vesselDetails, _ := h.detailsRepo.GetVesselDetails(c.Request.Context(), dealID)
	cargoDetails, _ := h.detailsRepo.GetCargoDetails(c.Request.Context(), dealID)

	c.JSON(http.StatusOK, gin.H{
		"deal":           deal,
		"participants":   participants,
		"vessel_details": vesselDetails,
		"cargo_details":  cargoDetails,
	})
}

// ---------- Attach charter party document ----------

type AttachDocumentRequest struct {
	DocumentID string `json:"document_id" binding:"required"`
}

func (h *Handler) handleAttachDocument(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	dealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deal ID"})
		return
	}
	if ok, _ := h.dealRepo.IsParticipant(c.Request.Context(), dealID, userID); !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req AttachDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	docID, err := uuid.Parse(req.DocumentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
		return
	}

	const q = `UPDATE shipman.deals SET document_id = $1, updated_at = NOW() WHERE id = $2`
	if _, err := db.Pool.ExecContext(c.Request.Context(), q, docID, dealID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to attach document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "document attached"})
}

// ---------- Vessel / Cargo detail upserts ----------

type UpsertVesselRequest struct {
	VesselName        *string  `json:"vessel_name"`
	IMONumber         *string  `json:"imo_number"`
	VesselType        *string  `json:"vessel_type"`
	FlagState         *string  `json:"flag_state"`
	DeadweightTonnage *float64 `json:"deadweight_tonnage"`
	GrossTonnage      *float64 `json:"gross_tonnage"`
	BuildYear         *int16   `json:"build_year"`
	ClassSociety      *string  `json:"class_society"`
	CurrentPosition   *string  `json:"current_position"`
	AvailableFrom     *string  `json:"available_from"` // RFC3339 date
	AskingRate        *float64 `json:"asking_rate"`
	AskingRateCurrency string  `json:"asking_rate_currency"`
	AskingRateType    string   `json:"asking_rate_type"`
	Notes             *string  `json:"notes"`
}

func (h *Handler) handleUpsertVesselDetails(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	dealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deal ID"})
		return
	}
	if ok, _ := h.dealRepo.IsParticipant(c.Request.Context(), dealID, userID); !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req UpsertVesselRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currency := req.AskingRateCurrency
	if currency == "" {
		currency = "USD"
	}
	rateType := req.AskingRateType
	if rateType == "" {
		rateType = "per_day"
	}

	d := &db.DealVesselDetails{
		DealID:             dealID,
		FilledBy:           userID,
		VesselName:         req.VesselName,
		IMONumber:          req.IMONumber,
		VesselType:         req.VesselType,
		FlagState:          req.FlagState,
		DeadweightTonnage:  req.DeadweightTonnage,
		GrossTonnage:       req.GrossTonnage,
		BuildYear:          req.BuildYear,
		ClassSociety:       req.ClassSociety,
		CurrentPosition:    req.CurrentPosition,
		AskingRate:         req.AskingRate,
		AskingRateCurrency: currency,
		AskingRateType:     rateType,
		Notes:              req.Notes,
	}
	if req.AvailableFrom != nil {
		t, err := time.Parse("2006-01-02", *req.AvailableFrom)
		if err == nil {
			d.AvailableFrom = &t
		}
	}

	if err := h.detailsRepo.UpsertVesselDetails(c.Request.Context(), d); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save vessel details"})
		return
	}
	c.JSON(http.StatusOK, d)
}

type UpsertCargoRequest struct {
	Commodity           *string  `json:"commodity"`
	Quantity            *float64 `json:"quantity"`
	QuantityUnit        string   `json:"quantity_unit"`
	LoadPort            *string  `json:"load_port"`
	DischargePort       *string  `json:"discharge_port"`
	LaycanFrom          *string  `json:"laycan_from"` // RFC3339 date
	LaycanTo            *string  `json:"laycan_to"`
	FreightIdea         *float64 `json:"freight_idea"`
	FreightCurrency     string   `json:"freight_currency"`
	FreightType         string   `json:"freight_type"`
	SpecialRequirements *string  `json:"special_requirements"`
	Notes               *string  `json:"notes"`
}

func (h *Handler) handleUpsertCargoDetails(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	dealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deal ID"})
		return
	}
	if ok, _ := h.dealRepo.IsParticipant(c.Request.Context(), dealID, userID); !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req UpsertCargoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	qUnit := req.QuantityUnit
	if qUnit == "" {
		qUnit = "MT"
	}
	fCurrency := req.FreightCurrency
	if fCurrency == "" {
		fCurrency = "USD"
	}
	fType := req.FreightType
	if fType == "" {
		fType = "per_mt"
	}

	d := &db.DealCargoDetails{
		DealID:              dealID,
		FilledBy:            userID,
		Commodity:           req.Commodity,
		Quantity:            req.Quantity,
		QuantityUnit:        qUnit,
		LoadPort:            req.LoadPort,
		DischargePort:       req.DischargePort,
		FreightIdea:         req.FreightIdea,
		FreightCurrency:     fCurrency,
		FreightType:         fType,
		SpecialRequirements: req.SpecialRequirements,
		Notes:               req.Notes,
	}
	if req.LaycanFrom != nil {
		t, err := time.Parse("2006-01-02", *req.LaycanFrom)
		if err == nil {
			d.LaycanFrom = &t
		}
	}
	if req.LaycanTo != nil {
		t, err := time.Parse("2006-01-02", *req.LaycanTo)
		if err == nil {
			d.LaycanTo = &t
		}
	}

	if err := h.detailsRepo.UpsertCargoDetails(c.Request.Context(), d); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save cargo details"})
		return
	}
	c.JSON(http.StatusOK, d)
}

// ---------- Public: invite preview ----------

func (h *Handler) handleGetInvitePreview(c *gin.Context) {
	token := c.Param("token")
	invite, err := h.dealRepo.GetInviteByToken(c.Request.Context(), token)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "invalid invite token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve invite"})
		return
	}

	if invite.UsedAt != nil {
		c.JSON(http.StatusGone, gin.H{"error": "invite has already been used"})
		return
	}

	if time.Now().After(invite.ExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": "invite has expired"})
		return
	}

	dealTitle := ""
	if d, err2 := h.dealRepo.Retrieve(c.Request.Context(), invite.DealID); err2 == nil {
		dealTitle = d.Title
	}

	c.JSON(http.StatusOK, gin.H{
		"token":         token,
		"role":          invite.Role,
		"deal_id":       invite.DealID,
		"deal_title":    dealTitle,
		"invited_email": invite.InvitedEmail,
		"expires_at":    invite.ExpiresAt,
	})
}

// ---------- Invite ----------

type CreateInviteRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=shipowner charterer broker"`
}

func (h *Handler) handleCreateInvite(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	inviterName := c.GetString("userFullName")

	dealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deal ID"})
		return
	}

	isParticipant, err := h.dealRepo.IsParticipant(c.Request.Context(), dealID, userID)
	if err != nil || !isParticipant {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req CreateInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invite := &db.DealInvite{
		DealID:       dealID,
		Role:         req.Role,
		InvitedEmail: req.Email,
		CreatedBy:    userID,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}

	if err := h.dealRepo.CreateInvite(c.Request.Context(), invite); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invite"})
		return
	}

	// Fetch deal title for the email
	dealTitle := ""
	if d, err2 := h.dealRepo.Retrieve(c.Request.Context(), dealID); err2 == nil {
		dealTitle = d.Title
	}

	// Send invite email (silently skipped when SMTP not configured)
	joinLink := fmt.Sprintf("%s/join?token=%s", h.appURL, invite.Token)
	go h.emailSvc.SendInvite(email.InviteEmailData{
		RecipientEmail: req.Email,
		RecipientRole:  req.Role,
		DealTitle:      dealTitle,
		InviterName:    inviterName,
		InviteLink:     joinLink,
	})

	c.JSON(http.StatusCreated, gin.H{
		"invite_token": invite.Token,
		"invite_link":  joinLink,
		"expires_at":   invite.ExpiresAt,
		"role":         invite.Role,
		"email_sent":   h.emailSvc.Enabled(),
	})
}

type JoinDealRequest struct {
	Token string `json:"token" binding:"required"`
}

func (h *Handler) handleJoinDeal(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req JoinDealRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invite, err := h.dealRepo.GetInviteByToken(c.Request.Context(), req.Token)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "invalid invite token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve invite"})
		return
	}

	if invite.UsedAt != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invite has already been used"})
		return
	}

	if time.Now().After(invite.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invite has expired"})
		return
	}

	isParticipant, _ := h.dealRepo.IsParticipant(c.Request.Context(), invite.DealID, userID)
	if isParticipant {
		c.JSON(http.StatusBadRequest, gin.H{"error": "you are already a participant in this deal"})
		return
	}

	now := time.Now()
	participant := &db.DealParticipant{
		DealID:    invite.DealID,
		UserID:    &userID,
		Role:      invite.Role,
		InvitedBy: &invite.CreatedBy,
		JoinedAt:  &now,
	}

	if err := h.dealRepo.AddParticipant(c.Request.Context(), participant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to join deal"})
		return
	}

	h.dealRepo.UseInvite(c.Request.Context(), req.Token, userID)

	deal, _ := h.dealRepo.Retrieve(c.Request.Context(), invite.DealID)

	c.JSON(http.StatusOK, gin.H{
		"message": "successfully joined deal",
		"deal":    deal,
		"role":    invite.Role,
	})
}

type CreateNegotiationRequest struct {
	ClauseType      string `json:"clause_type" binding:"required"`
	ClauseTitle     string `json:"clause_title" binding:"required"`
	OriginalContent string `json:"original_content" binding:"required"`
	SortOrder       int    `json:"sort_order"`
}

func (h *Handler) handleCreateNegotiation(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	dealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deal ID"})
		return
	}

	isParticipant, err := h.dealRepo.IsParticipant(c.Request.Context(), dealID, userID)
	if err != nil || !isParticipant {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req CreateNegotiationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	negotiation := &db.ClauseNegotiation{
		DealID:          dealID,
		ClauseType:      req.ClauseType,
		ClauseTitle:     req.ClauseTitle,
		OriginalContent: req.OriginalContent,
		Status:          "pending",
		SortOrder:       req.SortOrder,
	}

	if err := h.negRepo.CreateNegotiation(c.Request.Context(), negotiation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create negotiation"})
		return
	}

	c.JSON(http.StatusCreated, negotiation)
}

func (h *Handler) handleListNegotiations(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	dealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deal ID"})
		return
	}

	isParticipant, err := h.dealRepo.IsParticipant(c.Request.Context(), dealID, userID)
	if err != nil || !isParticipant {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	negotiations, err := h.negRepo.ListByDeal(c.Request.Context(), dealID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list negotiations"})
		return
	}

	if negotiations == nil {
		negotiations = []db.ClauseNegotiation{}
	}

	c.JSON(http.StatusOK, gin.H{"data": negotiations})
}

func (h *Handler) handleGetNegotiation(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	dealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deal ID"})
		return
	}

	isParticipant, err := h.dealRepo.IsParticipant(c.Request.Context(), dealID, userID)
	if err != nil || !isParticipant {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	negotiationID, err := uuid.Parse(c.Param("negotiationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid negotiation ID"})
		return
	}

	negotiation, err := h.negRepo.GetNegotiationWithProposals(c.Request.Context(), negotiationID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "negotiation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve negotiation"})
		return
	}

	c.JSON(http.StatusOK, negotiation)
}

type CreateProposalRequest struct {
	ProposedContent string  `json:"proposed_content" binding:"required"`
	Comment         *string `json:"comment"`
}

func (h *Handler) handleCreateProposal(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	dealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deal ID"})
		return
	}

	isParticipant, err := h.dealRepo.IsParticipant(c.Request.Context(), dealID, userID)
	if err != nil || !isParticipant {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	negotiationID, err := uuid.Parse(c.Param("negotiationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid negotiation ID"})
		return
	}

	var req CreateProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proposal := &db.ClauseProposal{
		NegotiationID:   negotiationID,
		ProposedBy:      userID,
		ProposedContent: req.ProposedContent,
		Comment:         req.Comment,
		Status:          "pending",
	}

	if err := h.negRepo.CreateProposal(c.Request.Context(), proposal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create proposal"})
		return
	}

	h.negRepo.UpdateStatus(c.Request.Context(), negotiationID, "countered")

	c.JSON(http.StatusCreated, proposal)
}

type UpdateProposalStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=accepted rejected"`
}

func (h *Handler) handleUpdateProposalStatus(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	dealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deal ID"})
		return
	}

	isParticipant, err := h.dealRepo.IsParticipant(c.Request.Context(), dealID, userID)
	if err != nil || !isParticipant {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	negotiationID, err := uuid.Parse(c.Param("negotiationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid negotiation ID"})
		return
	}

	proposalID, err := uuid.Parse(c.Param("proposalId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proposal ID"})
		return
	}

	var req UpdateProposalStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	if err := h.negRepo.UpdateProposalStatus(ctx, proposalID, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update proposal"})
		return
	}

	dealCompleted := false

	if req.Status == "accepted" {
		// Supersede all other pending proposals on this negotiation
		h.negRepo.SupersedeOtherProposals(ctx, negotiationID, proposalID)
		// Mark the negotiation itself as accepted
		h.negRepo.UpdateStatus(ctx, negotiationID, "accepted")

		// Check if ALL negotiations on the deal are now accepted
		allDone, err := h.negRepo.AllNegotiationsAccepted(ctx, dealID)
		if err == nil && allDone {
			h.dealRepo.UpdateStatus(ctx, dealID, "completed")
			dealCompleted = true
		}
	} else if req.Status == "rejected" {
		// Negotiation stays open for further proposals
		h.negRepo.UpdateStatus(ctx, negotiationID, "open")
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "proposal updated",
		"deal_completed": dealCompleted,
	})
}
