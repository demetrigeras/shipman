package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// VoyagePort mirrors shipman.voyage_ports rows.
type VoyagePort struct {
	ID              uuid.UUID  `json:"id"`
	VoyageID        uuid.UUID  `json:"voyage_id"`
	PortName        string     `json:"port_name"`
	PortCountry     *string    `json:"port_country,omitempty"`
	PortUNLocode    *string    `json:"port_unlocode,omitempty"`
	Latitude        *float64   `json:"latitude,omitempty"`
	Longitude       *float64   `json:"longitude,omitempty"`
	ArrivedAt       *time.Time `json:"arrived_at,omitempty"`
	DepartedAt      *time.Time `json:"departed_at,omitempty"`
	LaytimeHours    *float64   `json:"laytime_hours,omitempty"`
	CargoOperations *string    `json:"cargo_operations,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// VoyagePortService exposes CRUD behaviour.
type VoyagePortService interface {
	Create(ctx context.Context, vp *VoyagePort) error
	Retrieve(ctx context.Context, id uuid.UUID) (VoyagePort, error)
	ListByVoyage(ctx context.Context, voyageID uuid.UUID) ([]VoyagePort, error)
	Update(ctx context.Context, vp *VoyagePort) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// VoyagePortRepository implements VoyagePortService using Pool.
type VoyagePortRepository struct{}

// NewVoyagePortRepository returns repo.
func NewVoyagePortRepository() *VoyagePortRepository {
	return &VoyagePortRepository{}
}

// Create inserts a voyage port record.
func (repo *VoyagePortRepository) Create(ctx context.Context, vp *VoyagePort) error {
	const query = `
		INSERT INTO shipman.voyage_ports (
			voyage_id,
			port_name,
			port_country,
			port_unlocode,
			latitude,
			longitude,
			arrived_at,
			departed_at,
			laytime_hours,
			cargo_operations,
			notes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		RETURNING id, created_at, updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		vp.VoyageID,
		vp.PortName,
		nullableString(vp.PortCountry),
		nullableString(vp.PortUNLocode),
		nullableFloat(vp.Latitude),
		nullableFloat(vp.Longitude),
		nullableTime(vp.ArrivedAt),
		nullableTime(vp.DepartedAt),
		nullableFloat(vp.LaytimeHours),
		nullableString(vp.CargoOperations),
		nullableString(vp.Notes),
	).Scan(&vp.ID, &vp.CreatedAt, &vp.UpdatedAt)
}

// Retrieve fetches a voyage port by id.
func (repo *VoyagePortRepository) Retrieve(ctx context.Context, id uuid.UUID) (VoyagePort, error) {
	const query = `
		SELECT
			id,
		voyage_id,
			port_name,
			port_country,
			port_unlocode,
			latitude,
			longitude,
			arrived_at,
			departed_at,
			laytime_hours,
			cargo_operations,
			notes,
			created_at,
			updated_at
		FROM shipman.voyage_ports
		WHERE id = $1
	`

	var (
		vp        VoyagePort
		country   sql.NullString
		unlocode  sql.NullString
		lat       sql.NullFloat64
		lon       sql.NullFloat64
		arrival   sql.NullTime
		departure sql.NullTime
		laytime   sql.NullFloat64
		cargo     sql.NullString
		notes     sql.NullString
	)

	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&vp.ID,
		&vp.VoyageID,
		&vp.PortName,
		&country,
		&unlocode,
		&lat,
		&lon,
		&arrival,
		&departure,
		&laytime,
		&cargo,
		&notes,
		&vp.CreatedAt,
		&vp.UpdatedAt,
	)
	if err != nil {
		return VoyagePort{}, err
	}

	vp.PortCountry = stringPtr(country)
	vp.PortUNLocode = stringPtr(unlocode)
	vp.Latitude = floatPtr(lat)
	vp.Longitude = floatPtr(lon)
	vp.ArrivedAt = timePtr(arrival)
	vp.DepartedAt = timePtr(departure)
	vp.LaytimeHours = floatPtr(laytime)
	vp.CargoOperations = stringPtr(cargo)
	vp.Notes = stringPtr(notes)

	return vp, nil
}

// ListByVoyage returns all ports in order visited.
func (repo *VoyagePortRepository) ListByVoyage(ctx context.Context, voyageID uuid.UUID) ([]VoyagePort, error) {
	const query = `
		SELECT id, voyage_id, port_name, port_country, port_unlocode, latitude, longitude,
		       arrived_at, departed_at, laytime_hours, cargo_operations, notes, created_at, updated_at
		FROM shipman.voyage_ports
		WHERE voyage_id = $1
		ORDER BY arrived_at NULLS LAST, created_at
	`

	rows, err := Pool.QueryContext(ctx, query, voyageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ports []VoyagePort
	for rows.Next() {
		var (
			port      VoyagePort
			country   sql.NullString
			unlocode  sql.NullString
			lat       sql.NullFloat64
			lon       sql.NullFloat64
			arrival   sql.NullTime
			departure sql.NullTime
			laytime   sql.NullFloat64
			cargo     sql.NullString
			notes     sql.NullString
		)
		if err := rows.Scan(
			&port.ID,
			&port.VoyageID,
			&port.PortName,
			&country,
			&unlocode,
			&lat,
			&lon,
			&arrival,
			&departure,
			&laytime,
			&cargo,
			&notes,
			&port.CreatedAt,
			&port.UpdatedAt,
		); err != nil {
			return nil, err
		}
		port.PortCountry = stringPtr(country)
		port.PortUNLocode = stringPtr(unlocode)
		port.Latitude = floatPtr(lat)
		port.Longitude = floatPtr(lon)
		port.ArrivedAt = timePtr(arrival)
		port.DepartedAt = timePtr(departure)
		port.LaytimeHours = floatPtr(laytime)
		port.CargoOperations = stringPtr(cargo)
		port.Notes = stringPtr(notes)
		ports = append(ports, port)
	}
	return ports, rows.Err()
}

// Update modifies a port record.
func (repo *VoyagePortRepository) Update(ctx context.Context, vp *VoyagePort) error {
	const query = `
		UPDATE shipman.voyage_ports
		SET
			port_name = $2,
			port_country = $3,
			port_unlocode = $4,
			latitude = $5,
			longitude = $6,
			arrived_at = $7,
			departed_at = $8,
			laytime_hours = $9,
			cargo_operations = $10,
			notes = $11,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		vp.ID,
		vp.PortName,
		nullableString(vp.PortCountry),
		nullableString(vp.PortUNLocode),
		nullableFloat(vp.Latitude),
		nullableFloat(vp.Longitude),
		nullableTime(vp.ArrivedAt),
		nullableTime(vp.DepartedAt),
		nullableFloat(vp.LaytimeHours),
		nullableString(vp.CargoOperations),
		nullableString(vp.Notes),
	).Scan(&vp.UpdatedAt)
}

// Delete removes a voyage port.
func (repo *VoyagePortRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM shipman.voyage_ports WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id)
	return err
}
