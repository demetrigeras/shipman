package marketplace

import (
	"database/sql"
	"net/http"
	"strconv"

	"shipman/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	vesselRepo *db.VesselRepository
}

func NewHandler() *Handler {
	return &Handler{
		vesselRepo: db.NewVesselRepository(),
	}
}

func (h *Handler) AddRoutes(r *gin.RouterGroup) {
	r.GET("/vessels", h.handleListVessels)
	r.GET("/vessels/:id", h.handleGetVessel)
	r.POST("/vessels", h.handleCreateVessel)
	r.PUT("/vessels/:id", h.handleUpdateVessel)
	r.DELETE("/vessels/:id", h.handleDeleteVessel)
}

func (h *Handler) handleListVessels(c *gin.Context) {
	limit := 20
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	vessels, err := h.vesselRepo.List(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list vessels"})
		return
	}

	if vessels == nil {
		vessels = []db.Vessel{}
	}

	c.JSON(http.StatusOK, gin.H{"data": vessels})
}

func (h *Handler) handleGetVessel(c *gin.Context) {
	vesselID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vessel ID"})
		return
	}

	vessel, err := h.vesselRepo.Retrieve(c.Request.Context(), vesselID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "vessel not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve vessel"})
		return
	}

	c.JSON(http.StatusOK, vessel)
}

type CreateVesselRequest struct {
	Name              string   `json:"name" binding:"required"`
	IMONumber         *string  `json:"imo_number"`
	FlagState         *string  `json:"flag_state"`
	VesselType        *string  `json:"vessel_type"`
	CallSign          *string  `json:"call_sign"`
	DeadweightTonnage *float64 `json:"deadweight_tonnage"`
	GrossTonnage      *float64 `json:"gross_tonnage"`
	BuildYear         *int16   `json:"build_year"`
	Owner             *string  `json:"owner"`
	Notes             *string  `json:"notes"`
}

func (h *Handler) handleCreateVessel(c *gin.Context) {
	var req CreateVesselRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vessel := &db.Vessel{
		Name:              req.Name,
		IMONumber:         req.IMONumber,
		FlagState:         req.FlagState,
		VesselType:        req.VesselType,
		CallSign:          req.CallSign,
		DeadweightTonnage: req.DeadweightTonnage,
		GrossTonnage:      req.GrossTonnage,
		BuildYear:         req.BuildYear,
		Owner:             req.Owner,
		Notes:             req.Notes,
	}

	if err := h.vesselRepo.Create(c.Request.Context(), vessel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create vessel"})
		return
	}

	c.JSON(http.StatusCreated, vessel)
}

func (h *Handler) handleUpdateVessel(c *gin.Context) {
	vesselID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vessel ID"})
		return
	}

	existing, err := h.vesselRepo.Retrieve(c.Request.Context(), vesselID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "vessel not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve vessel"})
		return
	}

	var req CreateVesselRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing.Name = req.Name
	if req.IMONumber != nil {
		existing.IMONumber = req.IMONumber
	}
	if req.FlagState != nil {
		existing.FlagState = req.FlagState
	}
	if req.VesselType != nil {
		existing.VesselType = req.VesselType
	}
	if req.DeadweightTonnage != nil {
		existing.DeadweightTonnage = req.DeadweightTonnage
	}
	if req.GrossTonnage != nil {
		existing.GrossTonnage = req.GrossTonnage
	}
	if req.BuildYear != nil {
		existing.BuildYear = req.BuildYear
	}
	if req.Owner != nil {
		existing.Owner = req.Owner
	}
	if req.Notes != nil {
		existing.Notes = req.Notes
	}

	if err := h.vesselRepo.Update(c.Request.Context(), &existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update vessel"})
		return
	}

	c.JSON(http.StatusOK, existing)
}

func (h *Handler) handleDeleteVessel(c *gin.Context) {
	vesselID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vessel ID"})
		return
	}

	if err := h.vesselRepo.Delete(c.Request.Context(), vesselID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete vessel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "vessel deleted"})
}
