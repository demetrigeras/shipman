package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// ShipPosition mirrors shipman.ship_positions rows.
type ShipPosition struct {
	ID               uuid.UUID `json:"id"`
	VoyageID         uuid.UUID `json:"voyage_id"`
	RecordedAt       time.Time `json:"recorded_at"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	SpeedKnots       *float64  `json:"speed_knots,omitempty"`
	Heading          *float64  `json:"heading,omitempty"`
	DistanceLoggedNM *float64  `json:"distance_logged_nm,omitempty"`
	FuelRemainingMT  *float64  `json:"fuel_remaining_mt,omitempty"`
	Source           string    `json:"source"`
	Remarks          *string   `json:"remarks,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ShipPositionService exposes CRUD behaviour.
type ShipPositionService interface {
	Create(ctx context.Context, pos *ShipPosition) error
	Retrieve(ctx context.Context, id uuid.UUID) (ShipPosition, error)
	ListByVoyage(ctx context.Context, voyageID uuid.UUID, limit int) ([]ShipPosition, error)
	Update(ctx context.Context, pos *ShipPosition) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ShipPositionRepository implements ShipPositionService using Pool.
type ShipPositionRepository struct{}

// NewShipPositionRepository returns repo.
func NewShipPositionRepository() *ShipPositionRepository {
	return &ShipPositionRepository{}
}

// Create inserts a ship position.
func (repo *ShipPositionRepository) Create(ctx context.Context, pos *ShipPosition) error {
	const query = `
		INSERT INTO shipman.ship_positions (
			voyage_id,
			recorded_at,
			latitude,
			longitude,
			speed_knots,
			heading,
			distance_logged_nm,
			fuel_remaining_mt,
			source,
			remarks
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, COALESCE($9, 'manual'), $10
		)
		RETURNING id, source, created_at, updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		pos.VoyageID,
		pos.RecordedAt,
		pos.Latitude,
		pos.Longitude,
		nullableFloat(pos.SpeedKnots),
		nullableFloat(pos.Heading),
		nullableFloat(pos.DistanceLoggedNM),
		nullableFloat(pos.FuelRemainingMT),
		nullableString(&pos.Source),
		nullableString(pos.Remarks),
	).Scan(&pos.ID, &pos.Source, &pos.CreatedAt, &pos.UpdatedAt)
}

// Retrieve fetches a position by id.
func (repo *ShipPositionRepository) Retrieve(ctx context.Context, id uuid.UUID) (ShipPosition, error) {
	const query = `
		SELECT
			id,
			voyage_id,
			recorded_at,
			latitude,
			longitude,
			speed_knots,
			heading,
			distance_logged_nm,
			fuel_remaining_mt,
			source,
			remarks,
			created_at,
			updated_at
		FROM shipman.ship_positions
		WHERE id = $1
	`

	var (
		pos      ShipPosition
		speed    sql.NullFloat64
		heading  sql.NullFloat64
		distance sql.NullFloat64
		fuel     sql.NullFloat64
		source   sql.NullString
		remarks  sql.NullString
	)

	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&pos.ID,
		&pos.VoyageID,
		&pos.RecordedAt,
		&pos.Latitude,
		&pos.Longitude,
		&speed,
		&heading,
		&distance,
		&fuel,
		&source,
		&remarks,
		&pos.CreatedAt,
		&pos.UpdatedAt,
	)
	if err != nil {
		return ShipPosition{}, err
	}

	pos.SpeedKnots = floatPtr(speed)
	pos.Heading = floatPtr(heading)
	pos.DistanceLoggedNM = floatPtr(distance)
	pos.FuelRemainingMT = floatPtr(fuel)
	pos.Source = defaultString(source, "manual")
	pos.Remarks = stringPtr(remarks)

	return pos, nil
}

// ListByVoyage returns latest positions (limit if >0).
func (repo *ShipPositionRepository) ListByVoyage(ctx context.Context, voyageID uuid.UUID, limit int) ([]ShipPosition, error) {
	query := `
		SELECT id, voyage_id, recorded_at, latitude, longitude, speed_knots, heading,
		       distance_logged_nm, fuel_remaining_mt, source, remarks, created_at, updated_at
		FROM shipman.ship_positions
		WHERE voyage_id = $1
		ORDER BY recorded_at DESC
	`
	args := []any{voyageID}
	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
	}

	rows, err := Pool.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []ShipPosition
	for rows.Next() {
		var (
			pos      ShipPosition
			speed    sql.NullFloat64
			heading  sql.NullFloat64
			distance sql.NullFloat64
			fuel     sql.NullFloat64
			source   sql.NullString
			remarks  sql.NullString
		)
		if err := rows.Scan(
			&pos.ID,
			&pos.VoyageID,
			&pos.RecordedAt,
			&pos.Latitude,
			&pos.Longitude,
			&speed,
			&heading,
			&distance,
			&fuel,
			&source,
			&remarks,
			&pos.CreatedAt,
			&pos.UpdatedAt,
		); err != nil {
			return nil, err
		}
		pos.SpeedKnots = floatPtr(speed)
		pos.Heading = floatPtr(heading)
		pos.DistanceLoggedNM = floatPtr(distance)
		pos.FuelRemainingMT = floatPtr(fuel)
		pos.Source = defaultString(source, "manual")
		pos.Remarks = stringPtr(remarks)
		positions = append(positions, pos)
	}
	return positions, rows.Err()
}

// Update modifies a position row.
func (repo *ShipPositionRepository) Update(ctx context.Context, pos *ShipPosition) error {
	const query = `
		UPDATE shipman.ship_positions
		SET
			recorded_at = $2,
			latitude = $3,
			longitude = $4,
			speed_knots = $5,
			heading = $6,
			distance_logged_nm = $7,
			fuel_remaining_mt = $8,
			source = $9,
			remarks = $10,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		pos.ID,
		pos.RecordedAt,
		pos.Latitude,
		pos.Longitude,
		nullableFloat(pos.SpeedKnots),
		nullableFloat(pos.Heading),
		nullableFloat(pos.DistanceLoggedNM),
		nullableFloat(pos.FuelRemainingMT),
		pos.Source,
		nullableString(pos.Remarks),
	).Scan(&pos.UpdatedAt)
}

// Delete removes a position entry.
func (repo *ShipPositionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM shipman.ship_positions WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id)
	return err
}
