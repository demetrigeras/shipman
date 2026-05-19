package voyages

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"shipman/internal/ai"
	"shipman/internal/db"
	"shipman/internal/email"
)

type Handler struct {
	voyageRepo   *db.VoyageRepository
	positionRepo *db.ShipPositionRepository
	laytimeRepo  *db.LaytimeEntryRepository
	docRepo      *db.DocumentRepository
	userRepo     *db.UserRepository
	marineAPIKey string
	aiExtractor  ai.ClauseExtractor
	emailSvc     *email.Service
	appURL       string
}

func NewHandler(marineAPIKey, aiProvider, aiAPIKey, aiModel, aiBaseURL string, emailSvc *email.Service, appURL string) *Handler {
	var extractor ai.ClauseExtractor
	switch aiProvider {
	case "gemini":
		extractor = ai.NewGeminiExtractor(aiAPIKey, aiModel)
	default:
		extractor = ai.NewOpenAIExtractor(aiAPIKey, aiModel, aiBaseURL)
	}
	return &Handler{
		voyageRepo:   db.NewVoyageRepository(),
		positionRepo: db.NewShipPositionRepository(),
		laytimeRepo:  db.NewLaytimeEntryRepository(),
		docRepo:      db.NewDocumentRepository(),
		userRepo:     db.NewUserRepository(),
		marineAPIKey: marineAPIKey,
		aiExtractor:  extractor,
		emailSvc:     emailSvc,
		appURL:       appURL,
	}
}

func (h *Handler) AddRoutes(r *gin.RouterGroup) {
	// Voyages (fixtures)
	r.GET("", h.handleList)
	r.POST("", h.handleCreate)
	r.POST("/extract-terms-preview", h.handleExtractTermsPreview)
	r.POST("/join", h.handleJoinVoyage)
	r.GET("/:id", h.handleGet)
	r.PATCH("/:id", h.handleUpdate)
	r.DELETE("/:id", h.handleDelete)

	// Positions / tracking
	r.GET("/:id/positions", h.handleListPositions)
	r.POST("/:id/positions", h.handleAddPosition)
	r.GET("/:id/position/live", h.handleLivePosition)

	// Charter party document
	r.POST("/:id/attach-document", h.handleAttachDocument)
	r.POST("/:id/extract-terms", h.handleExtractTerms)

	// Invites
	r.POST("/:id/invite", h.handleCreateInvite)

	// Laytime
	r.GET("/:id/laytime", h.handleListLaytime)
	r.POST("/:id/laytime", h.handleAddLaytime)
	r.PATCH("/:id/laytime/:entryId", h.handleUpdateLaytime)
	r.DELETE("/:id/laytime/:entryId", h.handleDeleteLaytime)
	r.GET("/:id/laytime/summary", h.handleLaytimeSummary)
}

// AddPublicRoutes registers unauthenticated routes (invite preview).
func (h *Handler) AddPublicRoutes(r *gin.RouterGroup) {
	r.GET("/invite/:token", h.handlePreviewInvite)
}

// ---------- Voyage CRUD ----------

func (h *Handler) handleList(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	voyages, err := h.voyageRepo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list voyages"})
		return
	}
	if voyages == nil {
		voyages = []db.Voyage{}
	}
	c.JSON(http.StatusOK, voyages)
}

type UpsertVoyageRequest struct {
	VoyageNumber        *string    `json:"voyage_number"`
	CharterType         *string    `json:"charter_type"`
	VesselName          *string    `json:"vessel_name"`
	IMONumber           *string    `json:"imo_number"`
	VesselType          *string    `json:"vessel_type"`
	DWT                 *float64   `json:"dwt"`
	FlagState           *string    `json:"flag_state"`
	DeparturePort       *string    `json:"departure_port"`
	ArrivalPort         *string    `json:"arrival_port"`
	PlannedDeparture    *time.Time `json:"planned_departure_at"`
	PlannedArrival      *time.Time `json:"planned_arrival_at"`
	ActualDeparture     *time.Time `json:"actual_departure_at"`
	ActualArrival       *time.Time `json:"actual_arrival_at"`
	HireRate            *float64   `json:"hire_rate"`
	FreightRate         *float64   `json:"freight_rate"`
	CargoQuantity       *float64   `json:"cargo_quantity"`
	CargoType           *string    `json:"cargo_type"`
	LaytimeAllowedHours *float64   `json:"laytime_allowed_hours"`
	DemurrageRate       *float64   `json:"demurrage_rate"`
	DespatchRate        *float64   `json:"despatch_rate"`
	DemurrageCurrency   string     `json:"demurrage_currency"`
	PaymentFrequency    *string    `json:"payment_frequency"`
	FirstPaymentDate    *time.Time `json:"first_payment_date"`
	TotalContractValue  *float64   `json:"total_contract_value"`
	CommissionRate      *float64   `json:"commission_rate"`
	BunkerCost          *float64   `json:"bunker_cost"`
	PortCosts           *float64   `json:"port_costs"`
	InsuranceCost       *float64   `json:"insurance_cost"`
	CounterpartyName    *string    `json:"counterparty_name"`
	CounterpartyEmail   *string    `json:"counterparty_email"`
	Status              string     `json:"status"`
	Notes               *string    `json:"notes"`
	DealID              *string    `json:"deal_id"`
	ClearDocument       bool       `json:"clear_document"`
}

// normalizeDemurrageCurrency keeps DB column demurrage_currency CHAR(3) valid (AI often returns long strings).
func normalizeDemurrageCurrency(s string) string {
	s = strings.TrimSpace(strings.ToUpper(s))
	if len(s) == 3 {
		for _, r := range s {
			if r < 'A' || r > 'Z' {
				return "USD"
			}
		}
		return s
	}
	return "USD"
}

func (h *Handler) handleCreate(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req UpsertVoyageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	v := &db.Voyage{
		OwnerUserID:         &userID,
		VoyageNumber:        req.VoyageNumber,
		CharterType:         req.CharterType,
		VesselName:          req.VesselName,
		IMONumber:           req.IMONumber,
		VesselType:          req.VesselType,
		DWT:                 req.DWT,
		FlagState:           req.FlagState,
		DeparturePort:       req.DeparturePort,
		ArrivalPort:         req.ArrivalPort,
		PlannedDeparture:    req.PlannedDeparture,
		PlannedArrival:      req.PlannedArrival,
		ActualDeparture:     req.ActualDeparture,
		ActualArrival:       req.ActualArrival,
		HireRate:            req.HireRate,
		FreightRate:         req.FreightRate,
		CargoQuantity:       req.CargoQuantity,
		CargoType:           req.CargoType,
		LaytimeAllowedHours: req.LaytimeAllowedHours,
		DemurrageRate:       req.DemurrageRate,
		DespatchRate:        req.DespatchRate,
		DemurrageCurrency:   normalizeDemurrageCurrency(req.DemurrageCurrency),
		PaymentFrequency:    req.PaymentFrequency,
		FirstPaymentDate:    req.FirstPaymentDate,
		TotalContractValue:  req.TotalContractValue,
		CommissionRate:      req.CommissionRate,
		BunkerCost:          req.BunkerCost,
		PortCosts:           req.PortCosts,
		InsuranceCost:       req.InsuranceCost,
		CounterpartyName:    req.CounterpartyName,
		CounterpartyEmail:   req.CounterpartyEmail,
		Status:              req.Status,
		Notes:               req.Notes,
	}
	if req.DealID != nil {
		if parsed, err := uuid.Parse(*req.DealID); err == nil {
			v.DealID = &parsed
		}
	}

	if err := h.voyageRepo.Create(c.Request.Context(), v); err != nil {
		log.Printf("voyage create failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create voyage", "details": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *Handler) handleGet(c *gin.Context) {
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}
	v, err := h.voyageRepo.Retrieve(c.Request.Context(), voyageID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "voyage not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get voyage"})
		return
	}
	// Access check
	userID := c.MustGet("userID").(uuid.UUID)
	if v.OwnerUserID == nil || *v.OwnerUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) handleUpdate(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}

	existing, err := h.voyageRepo.Retrieve(c.Request.Context(), voyageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "voyage not found"})
		return
	}
	if existing.OwnerUserID == nil || *existing.OwnerUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req UpsertVoyageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Merge: only overwrite non-nil/non-zero fields
	if req.CharterType != nil { existing.CharterType = req.CharterType }
	if req.VoyageNumber != nil { existing.VoyageNumber = req.VoyageNumber }
	if req.VesselName != nil { existing.VesselName = req.VesselName }
	if req.IMONumber != nil { existing.IMONumber = req.IMONumber }
	if req.VesselType != nil { existing.VesselType = req.VesselType }
	if req.DWT != nil { existing.DWT = req.DWT }
	if req.FlagState != nil { existing.FlagState = req.FlagState }
	if req.DeparturePort != nil { existing.DeparturePort = req.DeparturePort }
	if req.ArrivalPort != nil { existing.ArrivalPort = req.ArrivalPort }
	if req.PlannedDeparture != nil { existing.PlannedDeparture = req.PlannedDeparture }
	if req.PlannedArrival != nil { existing.PlannedArrival = req.PlannedArrival }
	if req.ActualDeparture != nil { existing.ActualDeparture = req.ActualDeparture }
	if req.ActualArrival != nil { existing.ActualArrival = req.ActualArrival }
	if req.HireRate != nil { existing.HireRate = req.HireRate }
	if req.FreightRate != nil { existing.FreightRate = req.FreightRate }
	if req.CargoQuantity != nil { existing.CargoQuantity = req.CargoQuantity }
	if req.CargoType != nil { existing.CargoType = req.CargoType }
	if req.LaytimeAllowedHours != nil { existing.LaytimeAllowedHours = req.LaytimeAllowedHours }
	if req.DemurrageRate != nil { existing.DemurrageRate = req.DemurrageRate }
	if req.DespatchRate != nil { existing.DespatchRate = req.DespatchRate }
	if req.DemurrageCurrency != "" { existing.DemurrageCurrency = normalizeDemurrageCurrency(req.DemurrageCurrency) }
	if req.PaymentFrequency != nil { existing.PaymentFrequency = req.PaymentFrequency }
	if req.FirstPaymentDate != nil { existing.FirstPaymentDate = req.FirstPaymentDate }
	if req.TotalContractValue != nil { existing.TotalContractValue = req.TotalContractValue }
	if req.CommissionRate != nil { existing.CommissionRate = req.CommissionRate }
	if req.BunkerCost != nil { existing.BunkerCost = req.BunkerCost }
	if req.PortCosts != nil { existing.PortCosts = req.PortCosts }
	if req.InsuranceCost != nil { existing.InsuranceCost = req.InsuranceCost }
	if req.CounterpartyName != nil { existing.CounterpartyName = req.CounterpartyName }
	if req.CounterpartyEmail != nil { existing.CounterpartyEmail = req.CounterpartyEmail }
	if req.Status != "" { existing.Status = req.Status }
	if req.Notes != nil { existing.Notes = req.Notes }
	if req.ClearDocument { existing.DocumentID = nil }

	if err := h.voyageRepo.Update(c.Request.Context(), &existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update voyage"})
		return
	}
	c.JSON(http.StatusOK, existing)
}

func (h *Handler) handleDelete(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}
	existing, err := h.voyageRepo.Retrieve(c.Request.Context(), voyageID)
	if err != nil || existing.OwnerUserID == nil || *existing.OwnerUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	if err := h.voyageRepo.Delete(c.Request.Context(), voyageID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete voyage"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ---------- Position / Tracking ----------

func (h *Handler) handleListPositions(c *gin.Context) {
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}
	positions, err := h.positionRepo.ListByVoyage(c.Request.Context(), voyageID, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list positions"})
		return
	}
	if positions == nil {
		positions = []db.ShipPosition{}
	}
	c.JSON(http.StatusOK, positions)
}

type AddPositionRequest struct {
	RecordedAt       time.Time `json:"recorded_at" binding:"required"`
	Latitude         float64   `json:"latitude" binding:"required"`
	Longitude        float64   `json:"longitude" binding:"required"`
	SpeedKnots       *float64  `json:"speed_knots"`
	Heading          *float64  `json:"heading"`
	DistanceLoggedNM *float64  `json:"distance_logged_nm"`
	FuelRemainingMT  *float64  `json:"fuel_remaining_mt"`
	Remarks          *string   `json:"remarks"`
}

func (h *Handler) handleAddPosition(c *gin.Context) {
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}
	var req AddPositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pos := &db.ShipPosition{
		VoyageID:         voyageID,
		RecordedAt:       req.RecordedAt,
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		SpeedKnots:       req.SpeedKnots,
		Heading:          req.Heading,
		DistanceLoggedNM: req.DistanceLoggedNM,
		FuelRemainingMT:  req.FuelRemainingMT,
		Source:           "manual",
		Remarks:          req.Remarks,
	}
	if err := h.positionRepo.Create(c.Request.Context(), pos); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save position"})
		return
	}
	c.JSON(http.StatusCreated, pos)
}

// handleLivePosition fetches from MarineTraffic if configured, else returns latest manual position.
func (h *Handler) handleLivePosition(c *gin.Context) {
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}

	// Try to get the IMO number for this voyage
	v, err := h.voyageRepo.Retrieve(c.Request.Context(), voyageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "voyage not found"})
		return
	}

	// If MarineTraffic API key is configured and we have an IMO, try live lookup
	if h.marineAPIKey != "" && v.IMONumber != nil && *v.IMONumber != "" {
		pos, apiErr := fetchMarineTrafficPosition(*v.IMONumber, h.marineAPIKey)
		if apiErr == nil {
			pos.VoyageID = voyageID
			pos.Source = "ais"
			// Save it
			h.positionRepo.Create(c.Request.Context(), pos)
			c.JSON(http.StatusOK, gin.H{"source": "ais", "position": pos})
			return
		}
		// Fall through to latest manual on error
	}

	// Return latest manual position
	positions, err := h.positionRepo.ListByVoyage(c.Request.Context(), voyageID, 1)
	if err != nil || len(positions) == 0 {
		if h.marineAPIKey == "" && (v.IMONumber == nil || *v.IMONumber == "") {
			c.JSON(http.StatusOK, gin.H{"source": "none", "position": nil, "hint": "Add an IMO number and configure MarineTraffic API key for live tracking"})
		} else {
			c.JSON(http.StatusOK, gin.H{"source": "none", "position": nil})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"source": "manual", "position": positions[0]})
}

type marineTrafficError struct{ msg string }

func (e *marineTrafficError) Error() string { return e.msg }

// fetchMarineTrafficPosition calls the MarineTraffic API.
// TODO: implement real endpoint when subscription is ready:
// https://services.marinetraffic.com/api/exportvessel/v:8/{apiKey}/imo:{imoNumber}/protocol:jsono
func fetchMarineTrafficPosition(imoNumber, apiKey string) (*db.ShipPosition, error) {
	return nil, &marineTrafficError{msg: "MarineTraffic API not yet implemented"}
}

// ---------- Charter Party Document ----------

func (h *Handler) handleAttachDocument(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}

	existing, err := h.voyageRepo.Retrieve(c.Request.Context(), voyageID)
	if err != nil || existing.OwnerUserID == nil || *existing.OwnerUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req struct {
		DocumentID string `json:"document_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	docID, err := uuid.Parse(req.DocumentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
		return
	}

	if err := h.voyageRepo.AttachDocument(c.Request.Context(), voyageID, docID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to attach document"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "document attached", "document_id": docID})
}

// ExtractedTerms holds structured data parsed from a charter party.
type ExtractedTerms struct {
	VesselName          *string  `json:"vessel_name,omitempty"`
	IMONumber           *string  `json:"imo_number,omitempty"`
	VesselType          *string  `json:"vessel_type,omitempty"`
	DWT                 *float64 `json:"dwt,omitempty"`
	FlagState           *string  `json:"flag_state,omitempty"`
	HireRate            *float64 `json:"hire_rate,omitempty"`
	FreightRate         *float64 `json:"freight_rate,omitempty"`
	CargoType           *string  `json:"cargo_type,omitempty"`
	CargoQuantity       *float64 `json:"cargo_quantity,omitempty"`
	LoadPort            *string  `json:"load_port,omitempty"`
	DischargePort       *string  `json:"discharge_port,omitempty"`
	LaytimeAllowedHours *float64 `json:"laytime_allowed_hours,omitempty"`
	DemurrageRate       *float64 `json:"demurrage_rate,omitempty"`
	DespatchRate        *float64 `json:"despatch_rate,omitempty"`
	Currency            *string  `json:"currency,omitempty"`
	RawSummary          string   `json:"raw_summary,omitempty"`
}

// handleExtractTermsPreview runs charter-term extraction on a document without creating a voyage.
func (h *Handler) handleExtractTermsPreview(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	var req struct {
		DocumentID string `json:"document_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	docID, err := uuid.Parse(req.DocumentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
		return
	}
	doc, err := h.docRepo.Retrieve(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}
	if doc.UploadedBy != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	if doc.ExtractedText == nil || *doc.ExtractedText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document has no extracted text — wait for processing to finish"})
		return
	}
	if h.aiExtractor == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI not configured"})
		return
	}
	rawJSON, err := h.aiExtractor.ExtractTerms(c.Request.Context(), *doc.ExtractedText)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI extraction failed", "details": err.Error()})
		return
	}
	terms := parseTermsJSON(rawJSON)
	c.JSON(http.StatusOK, terms)
}

func (h *Handler) handleExtractTerms(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}

	existing, err := h.voyageRepo.Retrieve(c.Request.Context(), voyageID)
	if err != nil || existing.OwnerUserID == nil || *existing.OwnerUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Use document_id from voyage, or accept one in body
	var req struct {
		DocumentID string `json:"document_id"`
	}
	c.ShouldBindJSON(&req)

	var docID uuid.UUID
	if req.DocumentID != "" {
		docID, err = uuid.Parse(req.DocumentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
			return
		}
	} else if existing.DocumentID != nil {
		docID = *existing.DocumentID
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no document attached to this voyage"})
		return
	}

	doc, err := h.docRepo.Retrieve(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}
	if doc.ExtractedText == nil || *doc.ExtractedText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document has no extracted text — process it first in Documents"})
		return
	}

	rawJSON, err := h.aiExtractor.ExtractTerms(c.Request.Context(), *doc.ExtractedText)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "AI extraction failed",
			"details": err.Error(),
		})
		return
	}

	terms := parseTermsJSON(rawJSON)
	c.JSON(http.StatusOK, terms)
}

func parseTermsJSON(raw string) ExtractedTerms {
	var terms ExtractedTerms
	if err := json.Unmarshal([]byte(raw), &terms); err != nil {
		terms.RawSummary = fmt.Sprintf("Could not parse structured terms: %v\n\nRaw response:\n%s", err, raw)
	}
	return terms
}

// ---------- Laytime ----------

func (h *Handler) handleListLaytime(c *gin.Context) {
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}
	entries, err := h.laytimeRepo.ListByVoyage(c.Request.Context(), voyageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list laytime entries"})
		return
	}
	if entries == nil {
		entries = []db.LaytimeEntry{}
	}
	c.JSON(http.StatusOK, entries)
}

type LaytimeEntryRequest struct {
	PortName     string     `json:"port_name" binding:"required"`
	Activity     string     `json:"activity" binding:"required"`
	StartedAt    time.Time  `json:"started_at" binding:"required"`
	EndedAt      *time.Time `json:"ended_at"`
	HoursCounted *float64   `json:"hours_counted"`
	Remarks      *string    `json:"remarks"`
}

func (h *Handler) handleAddLaytime(c *gin.Context) {
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}

	// Get charter_detail_id from voyage (needed for laytime_entries FK)
	v, err := h.voyageRepo.Retrieve(c.Request.Context(), voyageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "voyage not found"})
		return
	}

	var req LaytimeEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Auto-calculate hours if start+end provided and hours not set
	hoursCounted := req.HoursCounted
	if hoursCounted == nil && req.EndedAt != nil {
		hrs := req.EndedAt.Sub(req.StartedAt).Hours()
		hoursCounted = &hrs
	}

	// Use a placeholder charter_detail_id if none (laytime_entries requires it due to old schema)
	var charterDetailID uuid.UUID
	if v.CharterDetailID != nil {
		charterDetailID = *v.CharterDetailID
	} else {
		// Use voyage ID as a stand-in UUID (same table, just needs a non-null value)
		// We'll relax this constraint in a future migration
		charterDetailID = voyageID
	}

	entry := &db.LaytimeEntry{
		CharterDetailID: charterDetailID,
		VoyageID:        &voyageID,
		PortName:        req.PortName,
		Activity:        req.Activity,
		StartedAt:       req.StartedAt,
		EndedAt:         req.EndedAt,
		HoursCounted:    hoursCounted,
		Remarks:         req.Remarks,
	}
	if err := h.laytimeRepo.Create(c.Request.Context(), entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add laytime entry"})
		return
	}
	c.JSON(http.StatusCreated, entry)
}

func (h *Handler) handleUpdateLaytime(c *gin.Context) {
	entryID, err := uuid.Parse(c.Param("entryId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entry ID"})
		return
	}
	existing, err := h.laytimeRepo.Retrieve(c.Request.Context(), entryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "entry not found"})
		return
	}
	var req LaytimeEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing.PortName = req.PortName
	existing.Activity = req.Activity
	existing.StartedAt = req.StartedAt
	existing.EndedAt = req.EndedAt
	existing.Remarks = req.Remarks
	if req.HoursCounted != nil {
		existing.HoursCounted = req.HoursCounted
	} else if req.EndedAt != nil {
		hrs := req.EndedAt.Sub(req.StartedAt).Hours()
		existing.HoursCounted = &hrs
	}
	if err := h.laytimeRepo.Update(c.Request.Context(), &existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update entry"})
		return
	}
	c.JSON(http.StatusOK, existing)
}

func (h *Handler) handleDeleteLaytime(c *gin.Context) {
	entryID, err := uuid.Parse(c.Param("entryId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entry ID"})
		return
	}
	if err := h.laytimeRepo.Delete(c.Request.Context(), entryID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete entry"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *Handler) handleLaytimeSummary(c *gin.Context) {
	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}
	summary, err := h.voyageRepo.CalcLaytime(c.Request.Context(), voyageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate laytime"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// ---------- Invite ----------

type CreateVoyageInviteRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=shipowner charterer broker"`
}

func (h *Handler) handleCreateInvite(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	inviterName := c.GetString("userFullName")

	voyageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid voyage ID"})
		return
	}

	existing, err := h.voyageRepo.Retrieve(c.Request.Context(), voyageID)
	if err != nil || existing.OwnerUserID == nil || *existing.OwnerUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req CreateVoyageInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invite := &db.VoyageInvite{
		VoyageID:     voyageID,
		Role:         req.Role,
		InvitedEmail: req.Email,
		CreatedBy:    userID,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}
	if err := h.voyageRepo.CreateInvite(c.Request.Context(), invite); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invite"})
		return
	}

	fixtureTitle := "Unnamed Fixture"
	if existing.VoyageNumber != nil {
		fixtureTitle = *existing.VoyageNumber
	} else if existing.VesselName != nil {
		fixtureTitle = *existing.VesselName
	}

	joinLink := fmt.Sprintf("%s/join?token=%s&type=voyage", h.appURL, invite.Token)
	go h.emailSvc.SendInvite(email.InviteEmailData{
		RecipientEmail: req.Email,
		RecipientRole:  req.Role,
		DealTitle:      fixtureTitle,
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

func (h *Handler) handlePreviewInvite(c *gin.Context) {
	token := c.Param("token")
	invite, err := h.voyageRepo.GetInviteByToken(c.Request.Context(), token)
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

	fixtureTitle := ""
	if v, err2 := h.voyageRepo.Retrieve(c.Request.Context(), invite.VoyageID); err2 == nil {
		if v.VoyageNumber != nil {
			fixtureTitle = *v.VoyageNumber
		} else if v.VesselName != nil {
			fixtureTitle = *v.VesselName
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"token":          invite.Token,
		"type":           "voyage",
		"role":           invite.Role,
		"voyage_id":      invite.VoyageID,
		"fixture_title":  fixtureTitle,
		"invited_email":  invite.InvitedEmail,
		"expires_at":     invite.ExpiresAt,
	})
}

func (h *Handler) handleJoinVoyage(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invite, err := h.voyageRepo.GetInviteByToken(c.Request.Context(), req.Token)
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

	// Get the joining user's details to store on the voyage
	joiningUser, err := h.userRepo.Retrieve(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	// Update voyage counterparty info with joining user's name + email
	if v, err2 := h.voyageRepo.Retrieve(c.Request.Context(), invite.VoyageID); err2 == nil {
		v.CounterpartyEmail = &joiningUser.Email
		v.CounterpartyName = &joiningUser.FullName
		if err3 := h.voyageRepo.Update(c.Request.Context(), &v); err3 != nil {
			log.Printf("handleJoinVoyage: failed to update counterparty: %v", err3)
		}
	}

	_ = h.voyageRepo.UseInvite(c.Request.Context(), req.Token, userID)

	c.JSON(http.StatusOK, gin.H{
		"message":    "joined voyage",
		"voyage_id":  invite.VoyageID,
		"role":       invite.Role,
	})
}
