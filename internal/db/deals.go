package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Deal struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	DocumentID  *uuid.UUID `json:"document_id,omitempty"`
	Status      string     `json:"status"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Canonical principal parties on the deal contract. These columns are
	// kept in sync with deal_participants by AddParticipant. They're nullable
	// because a deal may not have a counterparty (or broker) yet.
	ShipownerUserID *uuid.UUID `json:"shipowner_user_id,omitempty"`
	ChartererUserID *uuid.UUID `json:"charterer_user_id,omitempty"`
	BrokerUserID    *uuid.UUID `json:"broker_user_id,omitempty"`
}

type DealParticipant struct {
	ID          uuid.UUID  `json:"id"`
	DealID      uuid.UUID  `json:"deal_id"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	Role        string     `json:"role"`
	InvitedBy   *uuid.UUID `json:"invited_by,omitempty"`
	InviteEmail *string    `json:"invite_email,omitempty"`
	JoinedAt    *time.Time `json:"joined_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type DealInvite struct {
	ID           uuid.UUID  `json:"id"`
	DealID       uuid.UUID  `json:"deal_id"`
	Token        string     `json:"token"`
	Role         string     `json:"role"`
	InvitedEmail string     `json:"invited_email"`
	CreatedBy    uuid.UUID  `json:"created_by"`
	ExpiresAt    time.Time  `json:"expires_at"`
	UsedAt       *time.Time `json:"used_at,omitempty"`
	UsedBy       *uuid.UUID `json:"used_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type DealWithParticipants struct {
	Deal
	Participants []DealParticipantWithUser `json:"participants"`
}

type DealParticipantWithUser struct {
	DealParticipant
	User *User `json:"user,omitempty"`
}

type DealRepository struct{}

func NewDealRepository() *DealRepository {
	return &DealRepository{}
}

func (repo *DealRepository) Create(ctx context.Context, d *Deal) error {
	const query = `
		INSERT INTO shipman.deals (title, description, document_id, status, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	return Pool.QueryRowContext(ctx, query,
		d.Title, d.Description, d.DocumentID, d.Status, d.CreatedBy,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (repo *DealRepository) Retrieve(ctx context.Context, id uuid.UUID) (Deal, error) {
	const query = `
		SELECT id, title, description, document_id, status, created_by, created_at, updated_at,
		       shipowner_user_id, charterer_user_id, broker_user_id
		FROM shipman.deals
		WHERE id = $1
	`
	var d Deal
	var desc, docID sql.NullString
	var shipownerID, chartererID, brokerID sql.NullString

	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&d.ID, &d.Title, &desc, &docID, &d.Status, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
		&shipownerID, &chartererID, &brokerID,
	)
	if err != nil {
		return d, err
	}

	if desc.Valid {
		d.Description = &desc.String
	}
	if docID.Valid {
		uid, _ := uuid.Parse(docID.String)
		d.DocumentID = &uid
	}
	d.ShipownerUserID = nullStringToUUID(shipownerID)
	d.ChartererUserID = nullStringToUUID(chartererID)
	d.BrokerUserID = nullStringToUUID(brokerID)

	return d, nil
}

func (repo *DealRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]Deal, error) {
	const query = `
		SELECT DISTINCT d.id, d.title, d.description, d.document_id, d.status, d.created_by, d.created_at, d.updated_at,
		                d.shipowner_user_id, d.charterer_user_id, d.broker_user_id
		FROM shipman.deals d
		LEFT JOIN shipman.deal_participants dp ON d.id = dp.deal_id
		WHERE d.created_by = $1 OR dp.user_id = $1
		ORDER BY d.updated_at DESC
	`

	rows, err := Pool.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deals []Deal
	for rows.Next() {
		var d Deal
		var desc, docID sql.NullString
		var shipownerID, chartererID, brokerID sql.NullString

		if err := rows.Scan(
			&d.ID, &d.Title, &desc, &docID, &d.Status, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
			&shipownerID, &chartererID, &brokerID,
		); err != nil {
			return nil, err
		}

		if desc.Valid {
			d.Description = &desc.String
		}
		if docID.Valid {
			uid, _ := uuid.Parse(docID.String)
			d.DocumentID = &uid
		}
		d.ShipownerUserID = nullStringToUUID(shipownerID)
		d.ChartererUserID = nullStringToUUID(chartererID)
		d.BrokerUserID = nullStringToUUID(brokerID)

		deals = append(deals, d)
	}

	return deals, rows.Err()
}

// nullStringToUUID converts a sql.NullString holding a UUID into *uuid.UUID,
// returning nil when the column is NULL or unparseable.
func nullStringToUUID(s sql.NullString) *uuid.UUID {
	if !s.Valid {
		return nil
	}
	uid, err := uuid.Parse(s.String)
	if err != nil {
		return nil
	}
	return &uid
}

func (repo *DealRepository) AddParticipant(ctx context.Context, p *DealParticipant) error {
	const query = `
		INSERT INTO shipman.deal_participants (deal_id, user_id, role, invited_by, invite_email, joined_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	if err := Pool.QueryRowContext(ctx, query,
		p.DealID, p.UserID, p.Role, p.InvitedBy, p.InviteEmail, p.JoinedAt,
	).Scan(&p.ID, &p.CreatedAt); err != nil {
		return err
	}

	// Mirror the joined user into the matching principal-party column on the
	// deal so deals.shipowner_user_id / charterer_user_id / broker_user_id
	// always reflect the latest joined party for that role. Only update the
	// column when it's currently NULL — we don't want a second joiner to
	// silently displace an already-set party of record.
	if p.UserID != nil {
		_ = repo.setDealPartyIfEmpty(ctx, p.DealID, p.Role, *p.UserID)
	}
	return nil
}

// setDealPartyIfEmpty writes user_id into the deal's role column only when
// that column is currently NULL. Whitelists role to avoid SQL injection in
// the column-name interpolation.
func (repo *DealRepository) setDealPartyIfEmpty(ctx context.Context, dealID uuid.UUID, role string, userID uuid.UUID) error {
	col, ok := roleToColumn(role)
	if !ok {
		return nil
	}
	query := fmt.Sprintf(
		`UPDATE shipman.deals SET %s = $2 WHERE id = $1 AND %s IS NULL`,
		col, col,
	)
	_, err := Pool.ExecContext(ctx, query, dealID, userID)
	return err
}

// SetDealParty forces the role column to a specific user (or NULL when
// userID is the zero UUID). Use this when the deal creator explicitly
// re-assigns the party of record, independent of who has joined.
func (repo *DealRepository) SetDealParty(ctx context.Context, dealID uuid.UUID, role string, userID *uuid.UUID) error {
	col, ok := roleToColumn(role)
	if !ok {
		return fmt.Errorf("invalid role %q", role)
	}
	query := fmt.Sprintf(`UPDATE shipman.deals SET %s = $2 WHERE id = $1`, col)
	_, err := Pool.ExecContext(ctx, query, dealID, userID)
	return err
}

func roleToColumn(role string) (string, bool) {
	switch role {
	case "shipowner":
		return "shipowner_user_id", true
	case "charterer":
		return "charterer_user_id", true
	case "broker":
		return "broker_user_id", true
	default:
		return "", false
	}
}

func (repo *DealRepository) GetParticipants(ctx context.Context, dealID uuid.UUID) ([]DealParticipantWithUser, error) {
	const query = `
		SELECT dp.id, dp.deal_id, dp.user_id, dp.role, dp.invited_by, dp.invite_email, dp.joined_at, dp.created_at,
		       u.id, u.email, u.full_name, u.role
		FROM shipman.deal_participants dp
		LEFT JOIN shipman.users u ON dp.user_id = u.id
		WHERE dp.deal_id = $1
	`

	rows, err := Pool.QueryContext(ctx, query, dealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []DealParticipantWithUser
	for rows.Next() {
		var p DealParticipantWithUser
		var userID, invitedBy sql.NullString
		var inviteEmail sql.NullString
		var joinedAt sql.NullTime
		var uID, uEmail, uFullName, uRole sql.NullString

		if err := rows.Scan(
			&p.ID, &p.DealID, &userID, &p.Role, &invitedBy, &inviteEmail, &joinedAt, &p.CreatedAt,
			&uID, &uEmail, &uFullName, &uRole,
		); err != nil {
			return nil, err
		}

		if userID.Valid {
			uid, _ := uuid.Parse(userID.String)
			p.UserID = &uid
		}
		if invitedBy.Valid {
			uid, _ := uuid.Parse(invitedBy.String)
			p.InvitedBy = &uid
		}
		if inviteEmail.Valid {
			p.InviteEmail = &inviteEmail.String
		}
		if joinedAt.Valid {
			p.JoinedAt = &joinedAt.Time
		}
		if uID.Valid {
			uid, _ := uuid.Parse(uID.String)
			p.User = &User{
				ID:       uid,
				Email:    uEmail.String,
				FullName: uFullName.String,
				Role:     uRole.String,
			}
		}

		participants = append(participants, p)
	}

	return participants, rows.Err()
}

func (repo *DealRepository) IsParticipant(ctx context.Context, dealID, userID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM shipman.deals WHERE id = $1 AND created_by = $2
			UNION
			SELECT 1 FROM shipman.deal_participants WHERE deal_id = $1 AND user_id = $2
		)
	`
	var exists bool
	err := Pool.QueryRowContext(ctx, query, dealID, userID).Scan(&exists)
	return exists, err
}

func (repo *DealRepository) CreateInvite(ctx context.Context, i *DealInvite) error {
	token := make([]byte, 32)
	rand.Read(token)
	i.Token = hex.EncodeToString(token)

	const query = `
		INSERT INTO shipman.deal_invites (deal_id, token, role, invited_email, created_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	return Pool.QueryRowContext(ctx, query,
		i.DealID, i.Token, i.Role, i.InvitedEmail, i.CreatedBy, i.ExpiresAt,
	).Scan(&i.ID, &i.CreatedAt)
}

func (repo *DealRepository) GetInviteByToken(ctx context.Context, token string) (DealInvite, error) {
	const query = `
		SELECT id, deal_id, token, role, COALESCE(invited_email, ''), created_by, expires_at, used_at, used_by, created_at
		FROM shipman.deal_invites
		WHERE token = $1
	`
	var i DealInvite
	var usedAt sql.NullTime
	var usedBy sql.NullString

	err := Pool.QueryRowContext(ctx, query, token).Scan(
		&i.ID, &i.DealID, &i.Token, &i.Role, &i.InvitedEmail, &i.CreatedBy, &i.ExpiresAt, &usedAt, &usedBy, &i.CreatedAt,
	)
	if err != nil {
		return i, err
	}

	if usedAt.Valid {
		i.UsedAt = &usedAt.Time
	}
	if usedBy.Valid {
		uid, _ := uuid.Parse(usedBy.String)
		i.UsedBy = &uid
	}

	return i, nil
}

func (repo *DealRepository) UseInvite(ctx context.Context, token string, userID uuid.UUID) error {
	const query = `
		UPDATE shipman.deal_invites
		SET used_at = NOW(), used_by = $2
		WHERE token = $1
	`
	_, err := Pool.ExecContext(ctx, query, token, userID)
	return err
}

func (repo *DealRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	const query = `UPDATE shipman.deals SET status = $2 WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id, status)
	return err
}

// ListPendingInvites returns invites for a deal that haven't been used yet
// and haven't expired. Used by the UI to show "waiting on Alice@…" rows
// alongside the joined participants.
func (repo *DealRepository) ListPendingInvites(ctx context.Context, dealID uuid.UUID) ([]DealInvite, error) {
	const query = `
		SELECT id, deal_id, token, role, COALESCE(invited_email, ''), created_by, expires_at, used_at, used_by, created_at
		FROM shipman.deal_invites
		WHERE deal_id = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
		ORDER BY created_at DESC
	`
	rows, err := Pool.QueryContext(ctx, query, dealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []DealInvite
	for rows.Next() {
		var i DealInvite
		var usedAt sql.NullTime
		var usedBy sql.NullString
		if err := rows.Scan(
			&i.ID, &i.DealID, &i.Token, &i.Role, &i.InvitedEmail, &i.CreatedBy, &i.ExpiresAt, &usedAt, &usedBy, &i.CreatedAt,
		); err != nil {
			return nil, err
		}
		if usedAt.Valid {
			i.UsedAt = &usedAt.Time
		}
		if usedBy.Valid {
			uid, _ := uuid.Parse(usedBy.String)
			i.UsedBy = &uid
		}
		invites = append(invites, i)
	}
	return invites, rows.Err()
}

// CancelInvite hard-deletes an unused invite, scoped to the deal so a stolen
// invite ID can't delete a different deal's invite.
func (repo *DealRepository) CancelInvite(ctx context.Context, dealID, inviteID uuid.UUID) error {
	const query = `
		DELETE FROM shipman.deal_invites
		WHERE id = $1 AND deal_id = $2 AND used_at IS NULL
	`
	_, err := Pool.ExecContext(ctx, query, inviteID, dealID)
	return err
}
