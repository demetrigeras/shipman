package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Payment mirrors shipman.payments rows.
type Payment struct {
	ID              uuid.UUID  `json:"id"`
	CharterDetailID uuid.UUID  `json:"charter_detail_id"`
	VoyageID        *uuid.UUID `json:"voyage_id,omitempty"`
	Category        string     `json:"category"`
	DueDate         *time.Time `json:"due_date,omitempty"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	Amount          float64    `json:"amount"`
	Currency        string     `json:"currency"`
	Status          string     `json:"status"`
	PaymentMethod   *string    `json:"payment_method,omitempty"`
	Reference       *string    `json:"reference,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// PaymentService exposes CRUD behaviour for payments.
type PaymentService interface {
	Create(ctx context.Context, p *Payment) error
	Retrieve(ctx context.Context, id uuid.UUID) (Payment, error)
	ListByCharter(ctx context.Context, charterID uuid.UUID) ([]Payment, error)
	Update(ctx context.Context, p *Payment) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// PaymentRepository implements PaymentService using Pool.
type PaymentRepository struct{}

// NewPaymentRepository returns repository.
func NewPaymentRepository() *PaymentRepository {
	return &PaymentRepository{}
}

// Create inserts a payment.
func (repo *PaymentRepository) Create(ctx context.Context, p *Payment) error {
	const query = `
		INSERT INTO shipman.payments (
			charter_detail_id,
			voyage_id,
			category,
			due_date,
			paid_at,
			amount,
			currency,
			status,
			payment_method,
			reference,
			notes
		) VALUES (
			$1, $2, COALESCE($3, 'general'), $4, $5, $6,
			COALESCE($7, 'USD'), COALESCE($8, 'pending'), $9, $10, $11
		)
		RETURNING id, category, currency, status, created_at, updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		p.CharterDetailID,
		nullableUUID(p.VoyageID),
		nullableString(&p.Category),
		nullableTime(p.DueDate),
		nullableTime(p.PaidAt),
		p.Amount,
		nullableString(&p.Currency),
		nullableString(&p.Status),
		nullableString(p.PaymentMethod),
		nullableString(p.Reference),
		nullableString(p.Notes),
	).Scan(&p.ID, &p.Category, &p.Currency, &p.Status, &p.CreatedAt, &p.UpdatedAt)
}

// Retrieve fetches a payment by id.
func (repo *PaymentRepository) Retrieve(ctx context.Context, id uuid.UUID) (Payment, error) {
	const query = `
		SELECT
			id,
			charter_detail_id,
			voyage_id,
			category,
			due_date,
			paid_at,
			amount,
			currency,
			status,
			payment_method,
			reference,
			notes,
			created_at,
			updated_at
		FROM shipman.payments
		WHERE id = $1
	`

	var (
		payment   Payment
		rawVoy    sql.NullString
		due       sql.NullTime
		paid      sql.NullTime
		method    sql.NullString
		reference sql.NullString
		notes     sql.NullString
	)

	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&payment.ID,
		&payment.CharterDetailID,
		&rawVoy,
		&payment.Category,
		&due,
		&paid,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&method,
		&reference,
		&notes,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		return Payment{}, err
	}

	if rawVoy.Valid {
		if parsed, parseErr := uuid.Parse(rawVoy.String); parseErr == nil {
			payment.VoyageID = &parsed
		} else {
			return Payment{}, parseErr
		}
	}
	payment.DueDate = timePtr(due)
	payment.PaidAt = timePtr(paid)
	payment.PaymentMethod = stringPtr(method)
	payment.Reference = stringPtr(reference)
	payment.Notes = stringPtr(notes)

	return payment, nil
}

// ListByCharter returns payments for a charter.
func (repo *PaymentRepository) ListByCharter(ctx context.Context, charterID uuid.UUID) ([]Payment, error) {
	const query = `
		SELECT id, charter_detail_id, category, amount, status, due_date, paid_at, created_at, updated_at
		FROM shipman.payments
		WHERE charter_detail_id = $1
		ORDER BY due_date NULLS LAST, created_at DESC
	`

	rows, err := Pool.QueryContext(ctx, query, charterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []Payment
	for rows.Next() {
		var (
			payment Payment
			due     sql.NullTime
			paid    sql.NullTime
		)
		if err := rows.Scan(
			&payment.ID,
			&payment.CharterDetailID,
			&payment.Category,
			&payment.Amount,
			&payment.Status,
			&due,
			&paid,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		); err != nil {
			return nil, err
		}
		payment.DueDate = timePtr(due)
		payment.PaidAt = timePtr(paid)
		payments = append(payments, payment)
	}
	return payments, rows.Err()
}

// Update modifies payment fields.
func (repo *PaymentRepository) Update(ctx context.Context, p *Payment) error {
	const query = `
		UPDATE shipman.payments
		SET
			voyage_id = $2,
			category = $3,
			due_date = $4,
			paid_at = $5,
			amount = $6,
			currency = $7,
			status = $8,
			payment_method = $9,
			reference = $10,
			notes = $11,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		p.ID,
		nullableUUID(p.VoyageID),
		p.Category,
		nullableTime(p.DueDate),
		nullableTime(p.PaidAt),
		p.Amount,
		p.Currency,
		p.Status,
		nullableString(p.PaymentMethod),
		nullableString(p.Reference),
		nullableString(p.Notes),
	).Scan(&p.UpdatedAt)
}

// Delete removes a payment.
func (repo *PaymentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM shipman.payments WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id)
	return err
}
