package documents

import (
	"database/sql"
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"

	"shipman/internal/ai"
	"shipman/internal/db"
	"shipman/internal/processor"
	"shipman/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	docRepo   *db.DocumentRepository
	storage   storage.Storage
	processor *processor.Processor
	aiService ai.ClauseExtractor
}

func NewHandler(store storage.Storage, aiProvider, apiKey, aiModel, aiBaseURL string) *Handler {
	var aiService ai.ClauseExtractor
	if apiKey != "" {
		switch aiProvider {
		case "gemini":
			aiService = ai.NewGeminiExtractor(apiKey, aiModel)
		default:
			// "openai" or "deepseek" — both use OpenAI-compatible API
			aiService = ai.NewOpenAIExtractor(apiKey, aiModel, aiBaseURL)
		}
	}

	return &Handler{
		docRepo:   db.NewDocumentRepository(),
		storage:   store,
		processor: processor.NewProcessor(),
		aiService: aiService,
	}
}

func (h *Handler) AddRoutes(r *gin.RouterGroup) {
	r.POST("", h.handleUpload)
	r.GET("", h.handleList)
	r.GET("/:id", h.handleGet)
	// GET /:id/view is registered once in router.go (with query-token middleware for iframes)
	r.POST("/:id/process", h.handleProcess)
	r.POST("/:id/analyze", h.handleAnalyze)
	r.DELETE("/:id", h.handleDelete)
}

func (h *Handler) handleUpload(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file provided"})
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	allowedTypes := map[string]bool{
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"text/plain": true,
	}

	if !allowedTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file type not allowed. Supported: PDF, DOC, DOCX, TXT"})
		return
	}

	const maxSize = 50 * 1024 * 1024 // 50MB
	if header.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file too large. Maximum size is 50MB"})
		return
	}

	storagePath, err := h.storage.Save(header.Filename, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	var charterDetailID *uuid.UUID
	if charterIDStr := c.PostForm("charter_detail_id"); charterIDStr != "" {
		if id, err := uuid.Parse(charterIDStr); err == nil {
			charterDetailID = &id
		}
	}

	doc := &db.Document{
		CharterDetailID:  charterDetailID,
		UploadedBy:       userID.(uuid.UUID),
		Filename:         storagePath,
		OriginalFilename: header.Filename,
		ContentType:      contentType,
		FileSize:         header.Size,
		StoragePath:      storagePath,
		Status:           "uploaded",
	}

	if err := h.docRepo.Create(c.Request.Context(), doc); err != nil {
		h.storage.Delete(storagePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save document record"})
		return
	}

	c.JSON(http.StatusCreated, doc)
}

func (h *Handler) handleList(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

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

	docs, err := h.docRepo.ListByUser(c.Request.Context(), userID.(uuid.UUID), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list documents"})
		return
	}

	if docs == nil {
		docs = []db.Document{}
	}

	c.JSON(http.StatusOK, gin.H{"data": docs})
}

func (h *Handler) handleGet(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
		return
	}

	doc, err := h.docRepo.Retrieve(c.Request.Context(), docID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve document"})
		return
	}

	if doc.UploadedBy != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, doc)
}

func (h *Handler) handleProcess(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
		return
	}

	doc, err := h.docRepo.Retrieve(c.Request.Context(), docID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve document"})
		return
	}

	if doc.UploadedBy != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := h.docRepo.UpdateStatus(c.Request.Context(), docID, "processing"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	filePath := h.storage.GetFullPath(doc.StoragePath)
	text, err := h.processor.ExtractText(c.Request.Context(), filePath, doc.ContentType)
	if err != nil {
		h.docRepo.UpdateStatus(c.Request.Context(), docID, "failed")
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "failed to extract text",
			"details": err.Error(),
		})
		return
	}

	if err := h.docRepo.UpdateExtractedText(c.Request.Context(), docID, text); err != nil {
		h.docRepo.UpdateStatus(c.Request.Context(), docID, "failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save extracted text"})
		return
	}

	if err := h.docRepo.UpdateStatus(c.Request.Context(), docID, "processed"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	doc, _ = h.docRepo.Retrieve(c.Request.Context(), docID)
	c.JSON(http.StatusOK, doc)
}

func (h *Handler) handleAnalyze(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	if h.aiService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "AI service not configured",
			"details": "Add your API key to config/config.local.yaml under ai.openai_api_key. For free Gemini: https://aistudio.google.com/apikey",
		})
		return
	}

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
		return
	}

	doc, err := h.docRepo.Retrieve(c.Request.Context(), docID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve document"})
		return
	}

	if doc.UploadedBy != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Auto-process if text hasn't been extracted yet
	if doc.ExtractedText == nil || *doc.ExtractedText == "" {
		if err := h.docRepo.UpdateStatus(c.Request.Context(), docID, "processing"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
			return
		}
		filePath := h.storage.GetFullPath(doc.StoragePath)
		text, err := h.processor.ExtractText(c.Request.Context(), filePath, doc.ContentType)
		if err != nil {
			h.docRepo.UpdateStatus(c.Request.Context(), docID, "failed")
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   "failed to extract text from document",
				"details": err.Error(),
			})
			return
		}
		if err := h.docRepo.UpdateExtractedText(c.Request.Context(), docID, text); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save extracted text"})
			return
		}
		h.docRepo.UpdateStatus(c.Request.Context(), docID, "processed")
		doc, _ = h.docRepo.Retrieve(c.Request.Context(), docID)
	}

	result, err := h.aiService.ExtractClauses(c.Request.Context(), *doc.ExtractedText)
	if err != nil {
		// Log the full error to terminal for debugging
		log.Printf("AI analysis failed for doc %s: %v", docID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "AI analysis failed",
			"details": err.Error(),
		})
		return
	}

	analysisJSON, err := json.Marshal(result)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize analysis"})
		return
	}

	if err := h.docRepo.UpdateAIAnalysis(c.Request.Context(), docID, analysisJSON); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save analysis"})
		return
	}

	// Return updated document so frontend can refresh
	doc, _ = h.docRepo.Retrieve(c.Request.Context(), docID)

	c.JSON(http.StatusOK, gin.H{
		"document_id": docID,
		"analysis":    result,
		"document":    doc,
	})
}

func (h *Handler) handleDelete(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
		return
	}

	doc, err := h.docRepo.Retrieve(c.Request.Context(), docID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve document"})
		return
	}

	if doc.UploadedBy != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	h.storage.Delete(doc.StoragePath)

	if err := h.docRepo.Delete(c.Request.Context(), docID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "document deleted"})
}

// HandleView is the exported version for use outside the route group (e.g. with query-param token).
func (h *Handler) HandleView(c *gin.Context) { h.handleView(c) }

// handleView serves the raw document file (PDF, TXT, etc.) for inline preview.
func (h *Handler) handleView(c *gin.Context) {
	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
		return
	}

	doc, err := h.docRepo.Retrieve(c.Request.Context(), docID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve document"})
		return
	}

	f, err := h.storage.Get(doc.StoragePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found on disk"})
		return
	}
	defer f.Close()

	// Determine content type from file extension
	ext := filepath.Ext(doc.StoragePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Header("Content-Disposition", "inline; filename=\""+doc.OriginalFilename+"\"")
	c.DataFromReader(http.StatusOK, -1, contentType, f, nil)
}
