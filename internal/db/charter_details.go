package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// CharterDetail mirrors a row in shipman.charter_details.
type CharterDetail struct {
	ID                    uuid.UUID  `json:"id"`
	CreatedByUserID       *uuid.UUID `json:"created_by_user_id,omitempty"`
	Title                 string     `json:"title"`
	CharterReferenceCode  *string    `json:"charter_reference_code,omitempty"`
	VesselName            *string    `json:"vessel_name,omitempty"`
	CounterpartyName      *string    `json:"counterparty_name,omitempty"`
	Status                string     `json:"status"`
	StartDate             *time.Time `json:"start_date,omitempty"`
	EndDate               *time.Time `json:"end_date,omitempty"`
	LaytimeAllowanceHours *float64   `json:"laytime_allowance_hours,omitempty"`
	DemurrageRate         *float64   `json:"demurrage_rate,omitempty"`
	DemurrageCurrency     *string    `json:"demurrage_currency,omitempty"`
	FuelClause            *string    `json:"fuel_clause,omitempty"`
	PaymentTerms          *string    `json:"payment_terms,omitempty"`
	AIStatus              string     `json:"ai_status"`
	AIDocumentPath        *string    `json:"ai_document_path,omitempty"`
	AIExtractedTerms      []byte     `json:"ai_extracted_terms,omitempty"`
	LastReviewedAt        *time.Time `json:"last_reviewed_at,omitempty"`
	Notes                 *string    `json:"notes,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// CharterDetailService defines CRUD behaviour.
type CharterDetailService interface {
	Create(ctx context.Context, detail *CharterDetail) error
	Retrieve(ctx context.Context, id uuid.UUID) (CharterDetail, error)
	List(ctx context.Context, limit, offset int) ([]CharterDetail, error)
	Update(ctx context.Context, detail *CharterDetail) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// CharterDetailRepository implements CharterDetailService using the package Pool.
type CharterDetailRepository struct{}

// NewCharterDetailRepository returns a repository.
func NewCharterDetailRepository() *CharterDetailRepository {
	return &CharterDetailRepository{}
}

// Create inserts a charter detail row.
func (repo *CharterDetailRepository) Create(ctx context.Context, detail *CharterDetail) error {
	const query = `
		INSERT INTO shipman.charter_details (
			created_by_user_id,
			title,
			charter_reference_code,
			vessel_name,
			counterparty_name,
			status,
			start_date,
			end_date,
			laytime_allowance_hours,
			demurrage_rate,
			demurrage_currency,
			fuel_clause,
			payment_terms,
			ai_status,
			ai_document_path,
			ai_extracted_terms,
			last_reviewed_at,
			notes
		) VALUES (
			$1, $2, $3, $4, $5,
			COALESCE($6, 'draft'),
			$7, $8, $9, $10, $11,
			$12, $13, COALESCE($14, 'pending'),
			$15, $16, $17, $18
		)
		RETURNING id, status, ai_status, created_at, updated_at
	`

	status := detail.Status
	if status == "" {
		status = "draft"
	}
	aiStatus := detail.AIStatus
	if aiStatus == "" {
		aiStatus = "pending"
	}

	return Pool.QueryRowContext(
		ctx,
		query,
		nullableUUID(detail.CreatedByUserID),
		detail.Title,
		nullableString(detail.CharterReferenceCode),
		nullableString(detail.VesselName),
		nullableString(detail.CounterpartyName),
		status,
		nullableTime(detail.StartDate),
		nullableTime(detail.EndDate),
		nullableFloat(detail.LaytimeAllowanceHours),
		nullableFloat(detail.DemurrageRate),
		nullableString(detail.DemurrageCurrency),
		nullableString(detail.FuelClause),
		nullableString(detail.PaymentTerms),
		aiStatus,
		nullableString(detail.AIDocumentPath),
		nullableBytes(detail.AIExtractedTerms),
		nullableTime(detail.LastReviewedAt),
		nullableString(detail.Notes),
	).Scan(&detail.ID, &detail.Status, &detail.AIStatus, &detail.CreatedAt, &detail.UpdatedAt)
}

// Retrieve fetches a single charter detail.
func (repo *CharterDetailRepository) Retrieve(ctx context.Context, id uuid.UUID) (CharterDetail, error) {
	const query = `
		SELECT
			id,
			created_by_user_id,
			title,
			charter_reference_code,
			vessel_name,
			counterparty_name,
			status,
			start_date,
			end_date,
			laytime_allowance_hours,
			demurrage_rate,
			demurrage_currency,
			fuel_clause,
			payment_terms,
			ai_status,
			ai_document_path,
			ai_extracted_terms,
			last_reviewed_at,
			notes,
			created_at,
			updated_at
		FROM shipman.charter_details
		WHERE id = $1
	`

	var (
		detail     CharterDetail
		rawUserID  sql.NullString
		rawRef     sql.NullString
		rawVessel  sql.NullString
		rawCounter sql.NullString
		rawStatus  sql.NullString
		start      sql.NullTime
		end        sql.NullTime
		laytime    sql.NullFloat64
		demRate    sql.NullFloat64
		demCurr    sql.NullString
		fuel       sql.NullString
		payment    sql.NullString
		aiStatus   sql.NullString
		aiDoc      sql.NullString
		aiTerms    []byte
		lastRev    sql.NullTime
		notes      sql.NullString
	)

	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&detail.ID,
		&rawUserID,
		&detail.Title,
		&rawRef,
		&rawVessel,
		&rawCounter,
		&rawStatus,
		&start,
		&end,
		&laytime,
		&demRate,
		&demCurr,
		&fuel,
		&payment,
		&aiStatus,
		&aiDoc,
		&aiTerms,
		&lastRev,
		&notes,
		&detail.CreatedAt,
		&detail.UpdatedAt,
	)
	if err != nil {
		return CharterDetail{}, err
	}

	if rawUserID.Valid {
		if parsed, parseErr := uuid.Parse(rawUserID.String); parseErr == nil {
			detail.CreatedByUserID = &parsed
		} else {
			return CharterDetail{}, parseErr
		}
	}
	detail.CharterReferenceCode = stringPtr(rawRef)
	detail.VesselName = stringPtr(rawVessel)
	detail.CounterpartyName = stringPtr(rawCounter)
	detail.Status = defaultString(rawStatus, "draft")
	detail.StartDate = timePtr(start)
	detail.EndDate = timePtr(end)
	detail.LaytimeAllowanceHours = floatPtr(laytime)
	detail.DemurrageRate = floatPtr(demRate)
	detail.DemurrageCurrency = stringPtr(demCurr)
	detail.FuelClause = stringPtr(fuel)
	detail.PaymentTerms = stringPtr(payment)
	detail.AIStatus = defaultString(aiStatus, "pending")
	detail.AIDocumentPath = stringPtr(aiDoc)
	detail.AIExtractedTerms = bytesOrNil(aiTerms)
	detail.LastReviewedAt = timePtr(lastRev)
	detail.Notes = stringPtr(notes)

	return detail, nil
}

// List returns charter details ordered by most recent.
func (repo *CharterDetailRepository) List(ctx context.Context, limit, offset int) ([]CharterDetail, error) {
	const query = `
		SELECT id, title, status, created_at, updated_at
		FROM shipman.charter_details
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := Pool.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CharterDetail
	for rows.Next() {
		var detail CharterDetail
		if err := rows.Scan(&detail.ID, &detail.Title, &detail.Status, &detail.CreatedAt, &detail.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, detail)
	}
	return out, rows.Err()
}

// Update modifies editable fields of a charter detail.
func (repo *CharterDetailRepository) Update(ctx context.Context, detail *CharterDetail) error {
	const query = `
		UPDATE shipman.charter_details
		SET
			title = $2,
			charter_reference_code = $3,
			vessel_name = $4,
			counterparty_name = $5,
			status = $6,
			start_date = $7,
			end_date = $8,
			laytime_allowance_hours = $9,
			demurrage_rate = $10,
			demurrage_currency = $11,
			fuel_clause = $12,
			payment_terms = $13,
			ai_status = $14,
			ai_document_path = $15,
			ai_extracted_terms = $16,
			last_reviewed_at = $17,
			notes = $18,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		detail.ID,
		detail.Title,
		nullableString(detail.CharterReferenceCode),
		nullableString(detail.VesselName),
		nullableString(detail.CounterpartyName),
		detail.Status,
		nullableTime(detail.StartDate),
		nullableTime(detail.EndDate),
		nullableFloat(detail.LaytimeAllowanceHours),
		nullableFloat(detail.DemurrageRate),
		nullableString(detail.DemurrageCurrency),
		nullableString(detail.FuelClause),
		nullableString(detail.PaymentTerms),
		detail.AIStatus,
		nullableString(detail.AIDocumentPath),
		nullableBytes(detail.AIExtractedTerms),
		nullableTime(detail.LastReviewedAt),
		nullableString(detail.Notes),
	).Scan(&detail.UpdatedAt)
}

// Delete removes a charter detail.
func (repo *CharterDetailRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM shipman.charter_details WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id)
	return err
}
