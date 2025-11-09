package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Dispute mirrors shipman.disputes rows.
type Dispute struct {
	ID              uuid.UUID  `json:"id"`
	CharterDetailID uuid.UUID  `json:"charter_detail_id"`
	VoyageID        *uuid.UUID `json:"voyage_id,omitempty"`
	PaymentID       *uuid.UUID `json:"payment_id,omitempty"`
	LaytimeEntryID  *uuid.UUID `json:"laytime_entry_id,omitempty"`
	RaisedByOrgID   uuid.UUID  `json:"raised_by_org_id"`
	AssignedToOrgID *uuid.UUID `json:"assigned_to_org_id,omitempty"`
	Subject         string     `json:"subject"`
	Description     *string    `json:"description,omitempty"`
	ClaimedAmount   *float64   `json:"claimed_amount,omitempty"`
	Currency        *string    `json:"currency,omitempty"`
	Status          string     `json:"status"`
	ResolutionNotes *string    `json:"resolution_notes,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// DisputeService exposes CRUD behaviour.
type DisputeService interface {
	Create(ctx context.Context, d *Dispute) error
	Retrieve(ctx context.Context, id uuid.UUID) (Dispute, error)
	ListByCharter(ctx context.Context, charterID uuid.UUID) ([]Dispute, error)
	Update(ctx context.Context, d *Dispute) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// DisputeRepository implements DisputeService using Pool.
type DisputeRepository struct{}

// NewDisputeRepository returns repo.
func NewDisputeRepository() *DisputeRepository {
	return &DisputeRepository{}
}

// Create inserts dispute row.
func (repo *DisputeRepository) Create(ctx context.Context, d *Dispute) error {
	const query = `
		INSERT INTO shipman.disputes (
			charter_detail_id,
			voyage_id,
			payment_id,
			laytime_entry_id,
			raised_by_org_id,
			assigned_to_org_id,
			subject,
			description,
			claimed_amount,
			currency,
			status,
			resolution_notes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, COALESCE($11, 'open'), $12
		)
		RETURNING id, status, created_at, updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		d.CharterDetailID,
		nullableUUID(d.VoyageID),
		nullableUUID(d.PaymentID),
		nullableUUID(d.LaytimeEntryID),
		d.RaisedByOrgID,
		nullableUUID(d.AssignedToOrgID),
		d.Subject,
		nullableString(d.Description),
		nullableFloat(d.ClaimedAmount),
		nullableString(d.Currency),
		nullableString(&d.Status),
		nullableString(d.ResolutionNotes),
	).Scan(&d.ID, &d.Status, &d.CreatedAt, &d.UpdatedAt)
}

// Retrieve fetches dispute by id.
func (repo *DisputeRepository) Retrieve(ctx context.Context, id uuid.UUID) (Dispute, error) {
	const query = `
		SELECT
			id,
			charter_detail_id,
			voyage_id,
			payment_id,
			laytime_entry_id,
			raised_by_org_id,
			assigned_to_org_id,
			subject,
			description,
			claimed_amount,
			currency,
			status,
			resolution_notes,
			created_at,
			updated_at
		FROM shipman.disputes
		WHERE id = $1
	`

	var (
		dispute  Dispute
		voyage   sql.NullString
		payment  sql.NullString
		laytime  sql.NullString
		assigned sql.NullString
		desc     sql.NullString
		amount   sql.NullFloat64
		curr     sql.NullString
		status   sql.NullString
		notes    sql.NullString
	)

	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&dispute.ID,
		&dispute.CharterDetailID,
		&voyage,
		&payment,
		&laytime,
		&dispute.RaisedByOrgID,
		&assigned,
		&dispute.Subject,
		&desc,
		&amount,
		&curr,
		&status,
		&notes,
		&dispute.CreatedAt,
		&dispute.UpdatedAt,
	)
	if err != nil {
		return Dispute{}, err
	}

	dispute.VoyageID = uuidPtrNullable(voyage)
	dispute.PaymentID = uuidPtrNullable(payment)
	dispute.LaytimeEntryID = uuidPtrNullable(laytime)
	dispute.AssignedToOrgID = uuidPtrNullable(assigned)
	dispute.Description = stringPtr(desc)
	dispute.ClaimedAmount = floatPtr(amount)
	dispute.Currency = stringPtr(curr)
	dispute.Status = defaultString(status, "open")
	dispute.ResolutionNotes = stringPtr(notes)

	return dispute, nil
}

// ListByCharter returns disputes for a charter.
func (repo *DisputeRepository) ListByCharter(ctx context.Context, charterID uuid.UUID) ([]Dispute, error) {
	const query = `
		SELECT id, charter_detail_id, subject, status, claimed_amount, currency, created_at, updated_at
		FROM shipman.disputes
		WHERE charter_detail_id = $1
		ORDER BY created_at DESC
	`

	rows, err := Pool.QueryContext(ctx, query, charterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var disputes []Dispute
	for rows.Next() {
		var (
			dispute Dispute
			amount  sql.NullFloat64
			curr    sql.NullString
			status  sql.NullString
		)
		if err := rows.Scan(
			&dispute.ID,
			&dispute.CharterDetailID,
			&dispute.Subject,
			&status,
			&amount,
			&curr,
			&dispute.CreatedAt,
			&dispute.UpdatedAt,
		); err != nil {
			return nil, err
		}
		dispute.Status = defaultString(status, "open")
		dispute.ClaimedAmount = floatPtr(amount)
		dispute.Currency = stringPtr(curr)
		disputes = append(disputes, dispute)
	}
	return disputes, rows.Err()
}

// Update modifies dispute fields.
func (repo *DisputeRepository) Update(ctx context.Context, d *Dispute) error {
	const query = `
		UPDATE shipman.disputes
		SET
			voyage_id = $2,
			payment_id = $3,
			laytime_entry_id = $4,
			assigned_to_org_id = $5,
			subject = $6,
			description = $7,
			claimed_amount = $8,
			currency = $9,
			status = $10,
			resolution_notes = $11,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		d.ID,
		nullableUUID(d.VoyageID),
		nullableUUID(d.PaymentID),
		nullableUUID(d.LaytimeEntryID),
		nullableUUID(d.AssignedToOrgID),
		d.Subject,
		nullableString(d.Description),
		nullableFloat(d.ClaimedAmount),
		nullableString(d.Currency),
		d.Status,
		nullableString(d.ResolutionNotes),
	).Scan(&d.UpdatedAt)
}

// Delete removes a dispute.
func (repo *DisputeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM shipman.disputes WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id)
	return err
}
