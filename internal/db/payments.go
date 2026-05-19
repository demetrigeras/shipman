package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type VoyagePayment struct {
	ID                  uuid.UUID  `json:"id"`
	VoyageID            uuid.UUID  `json:"voyage_id"`
	CreatedBy           uuid.UUID  `json:"created_by"`
	PaymentType         string     `json:"payment_type"`
	Description         *string    `json:"description,omitempty"`
	Amount              float64    `json:"amount"`
	Currency            string     `json:"currency"`
	RecipientEmail      *string    `json:"recipient_email,omitempty"`
	RecipientWallet     *string    `json:"recipient_wallet,omitempty"`
	CoinsubSessionID    *string    `json:"coinsub_session_id,omitempty"`
	CoinsubPaymentID    *string    `json:"coinsub_payment_id,omitempty"`
	CoinsubAgreementID  *string    `json:"coinsub_agreement_id,omitempty"`
	CoinsubCheckoutURL  *string    `json:"coinsub_checkout_url,omitempty"`
	CoinsubTxHash       *string    `json:"coinsub_tx_hash,omitempty"`
	Status              string     `json:"status"`
	PaidAt              *time.Time `json:"paid_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type PaymentRepository struct{}

func NewPaymentRepository() *PaymentRepository {
	return &PaymentRepository{}
}

func (repo *PaymentRepository) Create(ctx context.Context, p *VoyagePayment) error {
	const query = `
		INSERT INTO shipman.voyage_payments
			(voyage_id, created_by, payment_type, description, amount, currency,
			 recipient_email, recipient_wallet, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`
	return Pool.QueryRowContext(ctx, query,
		p.VoyageID, p.CreatedBy, p.PaymentType, nullableString(p.Description),
		p.Amount, p.Currency,
		nullableString(p.RecipientEmail), nullableString(p.RecipientWallet),
		p.Status,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (repo *PaymentRepository) Retrieve(ctx context.Context, id uuid.UUID) (VoyagePayment, error) {
	const query = `
		SELECT id, voyage_id, created_by, payment_type, description, amount, currency,
		       recipient_email, recipient_wallet,
		       coinsub_session_id, coinsub_payment_id, coinsub_agreement_id,
		       coinsub_checkout_url, coinsub_tx_hash,
		       status, paid_at, created_at, updated_at
		FROM shipman.voyage_payments
		WHERE id = $1
	`
	var p VoyagePayment
	var desc, recEmail, recWallet sql.NullString
	var csSession, csPayment, csAgreement, csCheckout, csTxHash sql.NullString
	var paidAt sql.NullTime

	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.VoyageID, &p.CreatedBy, &p.PaymentType, &desc, &p.Amount, &p.Currency,
		&recEmail, &recWallet,
		&csSession, &csPayment, &csAgreement, &csCheckout, &csTxHash,
		&p.Status, &paidAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return p, err
	}
	p.Description = stringPtr(desc)
	p.RecipientEmail = stringPtr(recEmail)
	p.RecipientWallet = stringPtr(recWallet)
	p.CoinsubSessionID = stringPtr(csSession)
	p.CoinsubPaymentID = stringPtr(csPayment)
	p.CoinsubAgreementID = stringPtr(csAgreement)
	p.CoinsubCheckoutURL = stringPtr(csCheckout)
	p.CoinsubTxHash = stringPtr(csTxHash)
	if paidAt.Valid {
		p.PaidAt = &paidAt.Time
	}
	return p, nil
}

func (repo *PaymentRepository) ListByVoyage(ctx context.Context, voyageID uuid.UUID) ([]VoyagePayment, error) {
	const query = `
		SELECT id, voyage_id, created_by, payment_type, description, amount, currency,
		       recipient_email, recipient_wallet,
		       coinsub_session_id, coinsub_payment_id, coinsub_agreement_id,
		       coinsub_checkout_url, coinsub_tx_hash,
		       status, paid_at, created_at, updated_at
		FROM shipman.voyage_payments
		WHERE voyage_id = $1
		ORDER BY created_at DESC
	`
	rows, err := Pool.QueryContext(ctx, query, voyageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []VoyagePayment
	for rows.Next() {
		var p VoyagePayment
		var desc, recEmail, recWallet sql.NullString
		var csSession, csPayment, csAgreement, csCheckout, csTxHash sql.NullString
		var paidAt sql.NullTime

		if err := rows.Scan(
			&p.ID, &p.VoyageID, &p.CreatedBy, &p.PaymentType, &desc, &p.Amount, &p.Currency,
			&recEmail, &recWallet,
			&csSession, &csPayment, &csAgreement, &csCheckout, &csTxHash,
			&p.Status, &paidAt, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.Description = stringPtr(desc)
		p.RecipientEmail = stringPtr(recEmail)
		p.RecipientWallet = stringPtr(recWallet)
		p.CoinsubSessionID = stringPtr(csSession)
		p.CoinsubPaymentID = stringPtr(csPayment)
		p.CoinsubAgreementID = stringPtr(csAgreement)
		p.CoinsubCheckoutURL = stringPtr(csCheckout)
		p.CoinsubTxHash = stringPtr(csTxHash)
		if paidAt.Valid {
			p.PaidAt = &paidAt.Time
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}

func (repo *PaymentRepository) UpdateCoinsubSession(ctx context.Context, id uuid.UUID, sessionID, checkoutURL string) error {
	const query = `
		UPDATE shipman.voyage_payments
		SET coinsub_session_id = $2, coinsub_checkout_url = $3, status = 'pending'
		WHERE id = $1
	`
	_, err := Pool.ExecContext(ctx, query, id, sessionID, checkoutURL)
	return err
}

func (repo *PaymentRepository) MarkCompleted(ctx context.Context, sessionID, paymentID, txHash string) error {
	const query = `
		UPDATE shipman.voyage_payments
		SET status = 'completed', coinsub_payment_id = $2, coinsub_tx_hash = $3, paid_at = NOW()
		WHERE coinsub_session_id = $1
	`
	_, err := Pool.ExecContext(ctx, query, sessionID, paymentID, txHash)
	return err
}

// MarkCompletedByID marks a payment as completed using our internal payment UUID.
// Used by the webhook when the metadata contains the payment_id.
func (repo *PaymentRepository) MarkCompletedByID(ctx context.Context, id uuid.UUID, coinsubPaymentID, txHash, payerEmail string) error {
	const query = `
		UPDATE shipman.voyage_payments
		SET status = 'completed', coinsub_payment_id = $2, coinsub_tx_hash = $3,
		    paid_at = NOW(), recipient_email = COALESCE(NULLIF($4,''), recipient_email)
		WHERE id = $1
	`
	_, err := Pool.ExecContext(ctx, query, id, coinsubPaymentID, txHash, payerEmail)
	return err
}

func (repo *PaymentRepository) MarkFailed(ctx context.Context, sessionID string) error {
	const query = `
		UPDATE shipman.voyage_payments
		SET status = 'failed'
		WHERE coinsub_session_id = $1
	`
	_, err := Pool.ExecContext(ctx, query, sessionID)
	return err
}

// MarkFailedByID marks a payment as failed using our internal payment UUID.
func (repo *PaymentRepository) MarkFailedByID(ctx context.Context, id uuid.UUID) error {
	const query = `UPDATE shipman.voyage_payments SET status = 'failed' WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id)
	return err
}

func (repo *PaymentRepository) UpdateTransfer(ctx context.Context, id uuid.UUID, txHash string) error {
	const query = `
		UPDATE shipman.voyage_payments
		SET coinsub_tx_hash = $2, status = 'completed', paid_at = NOW()
		WHERE id = $1
	`
	_, err := Pool.ExecContext(ctx, query, id, txHash)
	return err
}

// MarkPaid manually marks a payment as completed (for testing / off-platform payments).
func (repo *PaymentRepository) MarkPaid(ctx context.Context, id uuid.UUID) error {
	const query = `
		UPDATE shipman.voyage_payments
		SET status = 'completed', paid_at = COALESCE(paid_at, NOW())
		WHERE id = $1 AND status != 'completed'
	`
	_, err := Pool.ExecContext(ctx, query, id)
	return err
}

func (repo *PaymentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := Pool.ExecContext(ctx, `DELETE FROM shipman.voyage_payments WHERE id = $1 AND status = 'draft'`, id)
	return err
}

func (repo *PaymentRepository) FindBySessionID(ctx context.Context, sessionID string) (VoyagePayment, error) {
	const query = `
		SELECT id, voyage_id, created_by, payment_type, description, amount, currency,
		       recipient_email, recipient_wallet,
		       coinsub_session_id, coinsub_payment_id, coinsub_agreement_id,
		       coinsub_checkout_url, coinsub_tx_hash,
		       status, paid_at, created_at, updated_at
		FROM shipman.voyage_payments
		WHERE coinsub_session_id = $1
	`
	var p VoyagePayment
	var desc, recEmail, recWallet sql.NullString
	var csSession, csPayment, csAgreement, csCheckout, csTxHash sql.NullString
	var paidAt sql.NullTime

	err := Pool.QueryRowContext(ctx, query, sessionID).Scan(
		&p.ID, &p.VoyageID, &p.CreatedBy, &p.PaymentType, &desc, &p.Amount, &p.Currency,
		&recEmail, &recWallet,
		&csSession, &csPayment, &csAgreement, &csCheckout, &csTxHash,
		&p.Status, &paidAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return p, err
	}
	p.Description = stringPtr(desc)
	p.RecipientEmail = stringPtr(recEmail)
	p.RecipientWallet = stringPtr(recWallet)
	p.CoinsubSessionID = stringPtr(csSession)
	p.CoinsubPaymentID = stringPtr(csPayment)
	p.CoinsubAgreementID = stringPtr(csAgreement)
	p.CoinsubCheckoutURL = stringPtr(csCheckout)
	p.CoinsubTxHash = stringPtr(csTxHash)
	if paidAt.Valid {
		p.PaidAt = &paidAt.Time
	}
	return p, nil
}
