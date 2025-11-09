package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// LaytimeEntry mirrors shipman.laytime_entries.
type LaytimeEntry struct {
	ID              uuid.UUID  `json:"id"`
	CharterDetailID uuid.UUID  `json:"charter_detail_id"`
	VoyageID        *uuid.UUID `json:"voyage_id,omitempty"`
	PortName        string     `json:"port_name"`
	Activity        string     `json:"activity"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	HoursCounted    *float64   `json:"hours_counted,omitempty"`
	Remarks         *string    `json:"remarks,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// LaytimeEntryService describes CRUD behaviour.
type LaytimeEntryService interface {
	Create(ctx context.Context, entry *LaytimeEntry) error
	Retrieve(ctx context.Context, id uuid.UUID) (LaytimeEntry, error)
	ListByVoyage(ctx context.Context, voyageID uuid.UUID) ([]LaytimeEntry, error)
	ListByCharter(ctx context.Context, charterID uuid.UUID) ([]LaytimeEntry, error)
	Update(ctx context.Context, entry *LaytimeEntry) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// LaytimeEntryRepository implements LaytimeEntryService using Pool.
type LaytimeEntryRepository struct{}

// NewLaytimeEntryRepository returns a repository.
func NewLaytimeEntryRepository() *LaytimeEntryRepository {
	return &LaytimeEntryRepository{}
}

// Create inserts a laytime entry.
func (repo *LaytimeEntryRepository) Create(ctx context.Context, entry *LaytimeEntry) error {
	const query = `
		INSERT INTO shipman.laytime_entries (
			charter_detail_id,
			voyage_id,
			port_name,
			activity,
			started_at,
			ended_at,
			hours_counted,
			remarks
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		RETURNING id, created_at, updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		entry.CharterDetailID,
		nullableUUID(entry.VoyageID),
		entry.PortName,
		entry.Activity,
		entry.StartedAt,
		nullableTime(entry.EndedAt),
		nullableFloat(entry.HoursCounted),
		nullableString(entry.Remarks),
	).Scan(&entry.ID, &entry.CreatedAt, &entry.UpdatedAt)
}

// Retrieve fetches an entry by id.
func (repo *LaytimeEntryRepository) Retrieve(ctx context.Context, id uuid.UUID) (LaytimeEntry, error) {
	const query = `
		SELECT
			id,
			charter_detail_id,
			voyage_id,
			port_name,
			activity,
			started_at,
			ended_at,
			hours_counted,
			remarks,
			created_at,
			updated_at
		FROM shipman.laytime_entries
		WHERE id = $1
	`

	var (
		entry   LaytimeEntry
		rawVoy  sql.NullString
		end     sql.NullTime
		hours   sql.NullFloat64
		remarks sql.NullString
	)

	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&entry.ID,
		&entry.CharterDetailID,
		&rawVoy,
		&entry.PortName,
		&entry.Activity,
		&entry.StartedAt,
		&end,
		&hours,
		&remarks,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)
	if err != nil {
		return LaytimeEntry{}, err
	}

	if rawVoy.Valid {
		if parsed, parseErr := uuid.Parse(rawVoy.String); parseErr == nil {
			entry.VoyageID = &parsed
		} else {
			return LaytimeEntry{}, parseErr
		}
	}
	entry.EndedAt = timePtr(end)
	entry.HoursCounted = floatPtr(hours)
	entry.Remarks = stringPtr(remarks)

	return entry, nil
}

// ListByVoyage returns entries for a voyage.
func (repo *LaytimeEntryRepository) ListByVoyage(ctx context.Context, voyageID uuid.UUID) ([]LaytimeEntry, error) {
	const query = `
		SELECT id, charter_detail_id, voyage_id, port_name, activity, started_at, ended_at, hours_counted, remarks, created_at, updated_at
		FROM shipman.laytime_entries
		WHERE voyage_id = $1
		ORDER BY started_at
	`

	rows, err := Pool.QueryContext(ctx, query, voyageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LaytimeEntry
	for rows.Next() {
		var (
			entry   LaytimeEntry
			rawVoy  sql.NullString
			end     sql.NullTime
			hours   sql.NullFloat64
			remarks sql.NullString
		)
		if err := rows.Scan(
			&entry.ID,
			&entry.CharterDetailID,
			&rawVoy,
			&entry.PortName,
			&entry.Activity,
			&entry.StartedAt,
			&end,
			&hours,
			&remarks,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if rawVoy.Valid {
			if parsed, parseErr := uuid.Parse(rawVoy.String); parseErr == nil {
				entry.VoyageID = &parsed
			} else {
				return nil, parseErr
			}
		}
		entry.EndedAt = timePtr(end)
		entry.HoursCounted = floatPtr(hours)
		entry.Remarks = stringPtr(remarks)
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// ListByCharter returns entries for a charter.
func (repo *LaytimeEntryRepository) ListByCharter(ctx context.Context, charterID uuid.UUID) ([]LaytimeEntry, error) {
	const query = `
		SELECT id, charter_detail_id, voyage_id, port_name, activity, started_at, ended_at, hours_counted, remarks, created_at, updated_at
		FROM shipman.laytime_entries
		WHERE charter_detail_id = $1
		ORDER BY started_at
	`

	rows, err := Pool.QueryContext(ctx, query, charterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LaytimeEntry
	for rows.Next() {
		var (
			entry   LaytimeEntry
			rawVoy  sql.NullString
			end     sql.NullTime
			hours   sql.NullFloat64
			remarks sql.NullString
		)
		if err := rows.Scan(
			&entry.ID,
			&entry.CharterDetailID,
			&rawVoy,
			&entry.PortName,
			&entry.Activity,
			&entry.StartedAt,
			&end,
			&hours,
			&remarks,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if rawVoy.Valid {
			if parsed, parseErr := uuid.Parse(rawVoy.String); parseErr == nil {
				entry.VoyageID = &parsed
			} else {
				return nil, parseErr
			}
		}
		entry.EndedAt = timePtr(end)
		entry.HoursCounted = floatPtr(hours)
		entry.Remarks = stringPtr(remarks)
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// Update modifies a laytime entry.
func (repo *LaytimeEntryRepository) Update(ctx context.Context, entry *LaytimeEntry) error {
	const query = `
		UPDATE shipman.laytime_entries
		SET
			voyage_id = $2,
			port_name = $3,
			activity = $4,
			started_at = $5,
			ended_at = $6,
			hours_counted = $7,
			remarks = $8,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		entry.ID,
		nullableUUID(entry.VoyageID),
		entry.PortName,
		entry.Activity,
		entry.StartedAt,
		nullableTime(entry.EndedAt),
		nullableFloat(entry.HoursCounted),
		nullableString(entry.Remarks),
	).Scan(&entry.UpdatedAt)
}

// Delete removes a laytime entry.
func (repo *LaytimeEntryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM shipman.laytime_entries WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id)
	return err
}
