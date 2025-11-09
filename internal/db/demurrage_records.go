package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// DemurrageRecord mirrors shipman.demurrage_records rows.
type DemurrageRecord struct {
	ID               uuid.UUID  `json:"id"`
	CharterDetailID  uuid.UUID  `json:"charter_detail_id"`
	VoyageID         *uuid.UUID `json:"voyage_id,omitempty"`
	LaytimeEntryID   *uuid.UUID `json:"laytime_entry_id,omitempty"`
	ClaimedHours     *float64   `json:"claimed_hours,omitempty"`
	ClaimedAmount    *float64   `json:"claimed_amount,omitempty"`
	Currency         string     `json:"currency"`
	Status           string     `json:"status"`
	Reference        *string    `json:"reference,omitempty"`
	SupportingDocURI *string    `json:"supporting_doc_uri,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// DemurrageRecordService exposes CRUD behaviour.
type DemurrageRecordService interface {
	Create(ctx context.Context, record *DemurrageRecord) error
	Retrieve(ctx context.Context, id uuid.UUID) (DemurrageRecord, error)
	ListByCharter(ctx context.Context, charterID uuid.UUID) ([]DemurrageRecord, error)
	Update(ctx context.Context, record *DemurrageRecord) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// DemurrageRecordRepository implements DemurrageRecordService using Pool.
type DemurrageRecordRepository struct{}

// NewDemurrageRecordRepository returns repository.
func NewDemurrageRecordRepository() *DemurrageRecordRepository {
	return &DemurrageRecordRepository{}
}

// Create inserts a demurrage record.
func (repo *DemurrageRecordRepository) Create(ctx context.Context, record *DemurrageRecord) error {
	const query = `
		INSERT INTO shipman.demurrage_records (
			charter_detail_id,
			voyage_id,
			laytime_entry_id,
			claimed_hours,
			claimed_amount,
			currency,
			status,
			reference,
			supporting_doc_uri,
			notes
		) VALUES (
			$1, $2, $3, $4, $5, COALESCE($6, 'USD'), COALESCE($7, 'draft'), $8, $9, $10
		)
		RETURNING id, currency, status, created_at, updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		record.CharterDetailID,
		nullableUUID(record.VoyageID),
		nullableUUID(record.LaytimeEntryID),
		nullableFloat(record.ClaimedHours),
		nullableFloat(record.ClaimedAmount),
		nullableString(&record.Currency),
		nullableString(&record.Status),
		nullableString(record.Reference),
		nullableString(record.SupportingDocURI),
		nullableString(record.Notes),
	).Scan(&record.ID, &record.Currency, &record.Status, &record.CreatedAt, &record.UpdatedAt)
}

// Retrieve fetches a demurrage record by id.
func (repo *DemurrageRecordRepository) Retrieve(ctx context.Context, id uuid.UUID) (DemurrageRecord, error) {
	const query = `
		SELECT
			id,
			charter_detail_id,
			voyage_id,
			laytime_entry_id,
			claimed_hours,
			claimed_amount,
			currency,
			status,
			reference,
			supporting_doc_uri,
			notes,
			created_at,
			updated_at
		FROM shipman.demurrage_records
		WHERE id = $1
	`

	var (
		record   DemurrageRecord
		voyage   sql.NullString
		laytime  sql.NullString
		hours    sql.NullFloat64
		amount   sql.NullFloat64
		currency sql.NullString
		status   sql.NullString
		ref      sql.NullString
		doc      sql.NullString
		notes    sql.NullString
	)

	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&record.ID,
		&record.CharterDetailID,
		&voyage,
		&laytime,
		&hours,
		&amount,
		&currency,
		&status,
		&ref,
		&doc,
		&notes,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return DemurrageRecord{}, err
	}

	record.VoyageID = uuidPtrNullable(voyage)
	record.LaytimeEntryID = uuidPtrNullable(laytime)
	record.ClaimedHours = floatPtr(hours)
	record.ClaimedAmount = floatPtr(amount)
	record.Currency = defaultString(currency, "USD")
	record.Status = defaultString(status, "draft")
	record.Reference = stringPtr(ref)
	record.SupportingDocURI = stringPtr(doc)
	record.Notes = stringPtr(notes)

	return record, nil
}

// ListByCharter returns demurrage records for a charter.
func (repo *DemurrageRecordRepository) ListByCharter(ctx context.Context, charterID uuid.UUID) ([]DemurrageRecord, error) {
	const query = `
		SELECT id, charter_detail_id, voyage_id, claimed_amount, status, created_at, updated_at
		FROM shipman.demurrage_records
		WHERE charter_detail_id = $1
		ORDER BY created_at DESC
	`

	rows, err := Pool.QueryContext(ctx, query, charterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []DemurrageRecord
	for rows.Next() {
		var (
			record DemurrageRecord
			voyage sql.NullString
			amount sql.NullFloat64
			status sql.NullString
		)
		if err := rows.Scan(
			&record.ID,
			&record.CharterDetailID,
			&voyage,
			&amount,
			&status,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, err
		}
		record.VoyageID = uuidPtrNullable(voyage)
		record.ClaimedAmount = floatPtr(amount)
		record.Status = defaultString(status, "draft")
		records = append(records, record)
	}
	return records, rows.Err()
}

// Update modifies a demurrage record.
func (repo *DemurrageRecordRepository) Update(ctx context.Context, record *DemurrageRecord) error {
	const query = `
		UPDATE shipman.demurrage_records
		SET
			voyage_id = $2,
			laytime_entry_id = $3,
			claimed_hours = $4,
			claimed_amount = $5,
			currency = $6,
			status = $7,
			reference = $8,
			supporting_doc_uri = $9,
			notes = $10,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		record.ID,
		nullableUUID(record.VoyageID),
		nullableUUID(record.LaytimeEntryID),
		nullableFloat(record.ClaimedHours),
		nullableFloat(record.ClaimedAmount),
		record.Currency,
		record.Status,
		nullableString(record.Reference),
		nullableString(record.SupportingDocURI),
		nullableString(record.Notes),
	).Scan(&record.UpdatedAt)
}

// Delete removes a demurrage record.
func (repo *DemurrageRecordRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM shipman.demurrage_records WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id)
	return err
}
