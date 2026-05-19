package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type ClauseNegotiation struct {
	ID              uuid.UUID `json:"id"`
	DealID          uuid.UUID `json:"deal_id"`
	ClauseType      string    `json:"clause_type"`
	ClauseTitle     string    `json:"clause_title"`
	OriginalContent string    `json:"original_content"`
	Status          string    `json:"status"`
	SortOrder       int       `json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ClauseProposal struct {
	ID              uuid.UUID `json:"id"`
	NegotiationID   uuid.UUID `json:"negotiation_id"`
	ProposedBy      uuid.UUID `json:"proposed_by"`
	ProposedContent string    `json:"proposed_content"`
	Comment         *string   `json:"comment,omitempty"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

type ClauseNegotiationWithProposals struct {
	ClauseNegotiation
	Proposals []ClauseProposalWithUser `json:"proposals"`
}

type ClauseProposalWithUser struct {
	ClauseProposal
	ProposedByUser *User `json:"proposed_by_user,omitempty"`
}

type NegotiationRepository struct{}

func NewNegotiationRepository() *NegotiationRepository {
	return &NegotiationRepository{}
}

func (repo *NegotiationRepository) CreateNegotiation(ctx context.Context, n *ClauseNegotiation) error {
	const query = `
		INSERT INTO shipman.clause_negotiations (deal_id, clause_type, clause_title, original_content, status, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (deal_id, clause_title) DO NOTHING
		RETURNING id, created_at, updated_at
	`
	err := Pool.QueryRowContext(ctx, query,
		n.DealID, n.ClauseType, n.ClauseTitle, n.OriginalContent, n.Status, n.SortOrder,
	).Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil // duplicate clause — silently skip
	}
	return err
}

func (repo *NegotiationRepository) GetNegotiation(ctx context.Context, id uuid.UUID) (ClauseNegotiation, error) {
	const query = `
		SELECT id, deal_id, clause_type, clause_title, original_content, status, sort_order, created_at, updated_at
		FROM shipman.clause_negotiations
		WHERE id = $1
	`
	var n ClauseNegotiation
	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&n.ID, &n.DealID, &n.ClauseType, &n.ClauseTitle, &n.OriginalContent, &n.Status, &n.SortOrder, &n.CreatedAt, &n.UpdatedAt,
	)
	return n, err
}

func (repo *NegotiationRepository) ListByDeal(ctx context.Context, dealID uuid.UUID) ([]ClauseNegotiation, error) {
	const query = `
		SELECT id, deal_id, clause_type, clause_title, original_content, status, sort_order, created_at, updated_at
		FROM shipman.clause_negotiations
		WHERE deal_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`

	rows, err := Pool.QueryContext(ctx, query, dealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var negotiations []ClauseNegotiation
	for rows.Next() {
		var n ClauseNegotiation
		if err := rows.Scan(&n.ID, &n.DealID, &n.ClauseType, &n.ClauseTitle, &n.OriginalContent, &n.Status, &n.SortOrder, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		negotiations = append(negotiations, n)
	}

	return negotiations, rows.Err()
}

func (repo *NegotiationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	const query = `UPDATE shipman.clause_negotiations SET status = $2 WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id, status)
	return err
}

func (repo *NegotiationRepository) CreateProposal(ctx context.Context, p *ClauseProposal) error {
	const query = `
		INSERT INTO shipman.clause_proposals (negotiation_id, proposed_by, proposed_content, comment, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	return Pool.QueryRowContext(ctx, query,
		p.NegotiationID, p.ProposedBy, p.ProposedContent, p.Comment, p.Status,
	).Scan(&p.ID, &p.CreatedAt)
}

func (repo *NegotiationRepository) GetProposals(ctx context.Context, negotiationID uuid.UUID) ([]ClauseProposalWithUser, error) {
	const query = `
		SELECT p.id, p.negotiation_id, p.proposed_by, p.proposed_content, p.comment, p.status, p.created_at,
		       u.id, u.email, u.full_name, u.role
		FROM shipman.clause_proposals p
		JOIN shipman.users u ON p.proposed_by = u.id
		WHERE p.negotiation_id = $1
		ORDER BY p.created_at ASC
	`

	rows, err := Pool.QueryContext(ctx, query, negotiationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proposals []ClauseProposalWithUser
	for rows.Next() {
		var p ClauseProposalWithUser
		var comment sql.NullString
		var uID, uEmail, uFullName, uRole string

		if err := rows.Scan(
			&p.ID, &p.NegotiationID, &p.ProposedBy, &p.ProposedContent, &comment, &p.Status, &p.CreatedAt,
			&uID, &uEmail, &uFullName, &uRole,
		); err != nil {
			return nil, err
		}

		if comment.Valid {
			p.Comment = &comment.String
		}

		uid, _ := uuid.Parse(uID)
		p.ProposedByUser = &User{
			ID:       uid,
			Email:    uEmail,
			FullName: uFullName,
			Role:     uRole,
		}

		proposals = append(proposals, p)
	}

	return proposals, rows.Err()
}

func (repo *NegotiationRepository) UpdateProposalStatus(ctx context.Context, id uuid.UUID, status string) error {
	const query = `UPDATE shipman.clause_proposals SET status = $2 WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id, status)
	return err
}

func (repo *NegotiationRepository) GetNegotiationWithProposals(ctx context.Context, id uuid.UUID) (ClauseNegotiationWithProposals, error) {
	n, err := repo.GetNegotiation(ctx, id)
	if err != nil {
		return ClauseNegotiationWithProposals{}, err
	}

	proposals, err := repo.GetProposals(ctx, id)
	if err != nil {
		return ClauseNegotiationWithProposals{}, err
	}

	return ClauseNegotiationWithProposals{
		ClauseNegotiation: n,
		Proposals:         proposals,
	}, nil
}

// SupersedeOtherProposals marks all other pending proposals on a negotiation as superseded
// when one proposal is accepted.
func (repo *NegotiationRepository) SupersedeOtherProposals(ctx context.Context, negotiationID, acceptedProposalID uuid.UUID) error {
	const query = `
		UPDATE shipman.clause_proposals
		SET status = 'superseded'
		WHERE negotiation_id = $1 AND id != $2 AND status = 'pending'
	`
	_, err := Pool.ExecContext(ctx, query, negotiationID, acceptedProposalID)
	return err
}

// AllNegotiationsAccepted returns true if every negotiation on a deal has status 'accepted'.
func (repo *NegotiationRepository) AllNegotiationsAccepted(ctx context.Context, dealID uuid.UUID) (bool, error) {
	const query = `
		SELECT COUNT(*) = 0
		FROM shipman.clause_negotiations
		WHERE deal_id = $1 AND status != 'accepted'
	`
	var allDone bool
	err := Pool.QueryRowContext(ctx, query, dealID).Scan(&allDone)
	return allDone, err
}

// GetNegotiationDealID returns the deal_id for a given negotiation.
func (repo *NegotiationRepository) GetNegotiationDealID(ctx context.Context, negotiationID uuid.UUID) (uuid.UUID, error) {
	const query = `SELECT deal_id FROM shipman.clause_negotiations WHERE id = $1`
	var dealID uuid.UUID
	err := Pool.QueryRowContext(ctx, query, negotiationID).Scan(&dealID)
	return dealID, err
}
