package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Document struct {
	ID               uuid.UUID       `json:"id"`
	CharterDetailID  *uuid.UUID      `json:"charter_detail_id,omitempty"`
	UploadedBy       uuid.UUID       `json:"uploaded_by"`
	Filename         string          `json:"filename"`
	OriginalFilename string          `json:"original_filename"`
	ContentType      string          `json:"content_type"`
	FileSize         int64           `json:"file_size"`
	StoragePath      string          `json:"-"`
	Status           string          `json:"status"`
	ExtractedText    *string         `json:"extracted_text,omitempty"`
	AIAnalysis       json.RawMessage `json:"ai_analysis,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type DocumentService interface {
	Create(ctx context.Context, d *Document) error
	Retrieve(ctx context.Context, id uuid.UUID) (Document, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Document, error)
	ListByCharter(ctx context.Context, charterID uuid.UUID) ([]Document, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateExtractedText(ctx context.Context, id uuid.UUID, text string) error
	UpdateAIAnalysis(ctx context.Context, id uuid.UUID, analysis json.RawMessage) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type DocumentRepository struct{}

func NewDocumentRepository() *DocumentRepository {
	return &DocumentRepository{}
}

func (repo *DocumentRepository) Create(ctx context.Context, d *Document) error {
	const query = `
		INSERT INTO shipman.documents (
			charter_detail_id, uploaded_by, filename, original_filename,
			content_type, file_size, storage_path, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	return Pool.QueryRowContext(ctx, query,
		d.CharterDetailID, d.UploadedBy, d.Filename, d.OriginalFilename,
		d.ContentType, d.FileSize, d.StoragePath, d.Status,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (repo *DocumentRepository) Retrieve(ctx context.Context, id uuid.UUID) (Document, error) {
	const query = `
		SELECT id, charter_detail_id, uploaded_by, filename, original_filename,
			   content_type, file_size, storage_path, status, extracted_text,
			   ai_analysis, created_at, updated_at
		FROM shipman.documents
		WHERE id = $1
	`

	var d Document
	var charterID sql.NullString
	var extractedText sql.NullString
	var aiAnalysis []byte

	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&d.ID, &charterID, &d.UploadedBy, &d.Filename, &d.OriginalFilename,
		&d.ContentType, &d.FileSize, &d.StoragePath, &d.Status, &extractedText,
		&aiAnalysis, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return d, err
	}

	if charterID.Valid {
		uid, _ := uuid.Parse(charterID.String)
		d.CharterDetailID = &uid
	}
	if extractedText.Valid {
		d.ExtractedText = &extractedText.String
	}
	if aiAnalysis != nil {
		d.AIAnalysis = aiAnalysis
	}

	return d, nil
}

func (repo *DocumentRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Document, error) {
	const query = `
		SELECT id, charter_detail_id, uploaded_by, filename, original_filename,
			   content_type, file_size, storage_path, status, extracted_text,
			   ai_analysis, created_at, updated_at
		FROM shipman.documents
		WHERE uploaded_by = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := Pool.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDocuments(rows)
}

func (repo *DocumentRepository) ListByCharter(ctx context.Context, charterID uuid.UUID) ([]Document, error) {
	const query = `
		SELECT id, charter_detail_id, uploaded_by, filename, original_filename,
			   content_type, file_size, storage_path, status, extracted_text,
			   ai_analysis, created_at, updated_at
		FROM shipman.documents
		WHERE charter_detail_id = $1
		ORDER BY created_at DESC
	`

	rows, err := Pool.QueryContext(ctx, query, charterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDocuments(rows)
}

func scanDocuments(rows *sql.Rows) ([]Document, error) {
	var docs []Document
	for rows.Next() {
		var d Document
		var charterID sql.NullString
		var extractedText sql.NullString
		var aiAnalysis []byte

		if err := rows.Scan(
			&d.ID, &charterID, &d.UploadedBy, &d.Filename, &d.OriginalFilename,
			&d.ContentType, &d.FileSize, &d.StoragePath, &d.Status, &extractedText,
			&aiAnalysis, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if charterID.Valid {
			uid, _ := uuid.Parse(charterID.String)
			d.CharterDetailID = &uid
		}
		if extractedText.Valid {
			d.ExtractedText = &extractedText.String
		}
		if aiAnalysis != nil {
			d.AIAnalysis = aiAnalysis
		}

		docs = append(docs, d)
	}
	return docs, rows.Err()
}

func (repo *DocumentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	const query = `UPDATE shipman.documents SET status = $2 WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id, status)
	return err
}

func (repo *DocumentRepository) UpdateExtractedText(ctx context.Context, id uuid.UUID, text string) error {
	const query = `UPDATE shipman.documents SET extracted_text = $2 WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id, text)
	return err
}

func (repo *DocumentRepository) UpdateAIAnalysis(ctx context.Context, id uuid.UUID, analysis json.RawMessage) error {
	const query = `UPDATE shipman.documents SET ai_analysis = $2 WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id, analysis)
	return err
}

func (repo *DocumentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM shipman.documents WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id)
	return err
}
