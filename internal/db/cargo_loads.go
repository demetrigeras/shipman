package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// CargoLoad mirrors shipman.cargo_loads rows.
type CargoLoad struct {
	ID            uuid.UUID `json:"id"`
	VoyageID      uuid.UUID `json:"voyage_id"`
	LoadPort      *string   `json:"load_port,omitempty"`
	DischargePort *string   `json:"discharge_port,omitempty"`
	Commodity     *string   `json:"commodity,omitempty"`
	Quantity      *float64  `json:"quantity,omitempty"`
	Unit          *string   `json:"unit,omitempty"`
	StowagePlan   []byte    `json:"stowage_plan,omitempty"`
	Hazardous     *bool     `json:"hazardous,omitempty"`
	Notes         *string   `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CargoLoadService exposes CRUD behaviour.
type CargoLoadService interface {
	Create(ctx context.Context, load *CargoLoad) error
	Retrieve(ctx context.Context, id uuid.UUID) (CargoLoad, error)
	ListByVoyage(ctx context.Context, voyageID uuid.UUID) ([]CargoLoad, error)
	Update(ctx context.Context, load *CargoLoad) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// CargoLoadRepository implements CargoLoadService using Pool.
type CargoLoadRepository struct{}

// NewCargoLoadRepository returns a repository.
func NewCargoLoadRepository() *CargoLoadRepository {
	return &CargoLoadRepository{}
}

// Create inserts a cargo load row.
func (repo *CargoLoadRepository) Create(ctx context.Context, load *CargoLoad) error {
	const query = `
		INSERT INTO shipman.cargo_loads (
			voyage_id,
			load_port,
			discharge_port,
			commodity,
			quantity,
			unit,
			stowage_plan,
			hazardous,
			notes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		RETURNING id, created_at, updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		load.VoyageID,
		nullableString(load.LoadPort),
		nullableString(load.DischargePort),
		nullableString(load.Commodity),
		nullableFloat(load.Quantity),
		nullableString(load.Unit),
		nullableBytes(load.StowagePlan),
		nullableBool(load.Hazardous),
		nullableString(load.Notes),
	).Scan(&load.ID, &load.CreatedAt, &load.UpdatedAt)
}

// Retrieve fetches a cargo load by id.
func (repo *CargoLoadRepository) Retrieve(ctx context.Context, id uuid.UUID) (CargoLoad, error) {
	const query = `
		SELECT
			id,
			voyage_id,
			load_port,
			discharge_port,
			commodity,
			quantity,
			unit,
			stowage_plan,
			hazardous,
			notes,
			created_at,
			updated_at
		FROM shipman.cargo_loads
		WHERE id = $1
	`

	var (
		load      CargoLoad
		loadPort  sql.NullString
		discharge sql.NullString
		commodity sql.NullString
		quantity  sql.NullFloat64
		unit      sql.NullString
		stowage   []byte
		hazardous sql.NullBool
		notes     sql.NullString
	)

	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&load.ID,
		&load.VoyageID,
		&loadPort,
		&discharge,
		&commodity,
		&quantity,
		&unit,
		&stowage,
		&hazardous,
		&notes,
		&load.CreatedAt,
		&load.UpdatedAt,
	)
	if err != nil {
		return CargoLoad{}, err
	}

	load.LoadPort = stringPtr(loadPort)
	load.DischargePort = stringPtr(discharge)
	load.Commodity = stringPtr(commodity)
	load.Quantity = floatPtr(quantity)
	load.Unit = stringPtr(unit)
	load.StowagePlan = bytesOrNil(stowage)
	if hazardous.Valid {
		val := hazardous.Bool
		load.Hazardous = &val
	}
	load.Notes = stringPtr(notes)

	return load, nil
}

// ListByVoyage returns cargo loads for a voyage.
func (repo *CargoLoadRepository) ListByVoyage(ctx context.Context, voyageID uuid.UUID) ([]CargoLoad, error) {
	const query = `
		SELECT id, voyage_id, commodity, quantity, unit, created_at, updated_at
		FROM shipman.cargo_loads
		WHERE voyage_id = $1
		ORDER BY created_at DESC
	`

	rows, err := Pool.QueryContext(ctx, query, voyageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var loads []CargoLoad
	for rows.Next() {
		var (
			load      CargoLoad
			commodity sql.NullString
			quantity  sql.NullFloat64
			unit      sql.NullString
		)
		if err := rows.Scan(
			&load.ID,
			&load.VoyageID,
			&commodity,
			&quantity,
			&unit,
			&load.CreatedAt,
			&load.UpdatedAt,
		); err != nil {
			return nil, err
		}
		load.Commodity = stringPtr(commodity)
		load.Quantity = floatPtr(quantity)
		load.Unit = stringPtr(unit)
		loads = append(loads, load)
	}
	return loads, rows.Err()
}

// Update modifies a cargo load.
func (repo *CargoLoadRepository) Update(ctx context.Context, load *CargoLoad) error {
	const query = `
		UPDATE shipman.cargo_loads
		SET
			load_port = COALESCE($2, load_port),
			discharge_port = COALESCE($3, discharge_port),
			commodity = COALESCE($4, commodity),
			quantity = COALESCE($5, quantity),
			unit = COALESCE($6, unit),
			stowage_plan = COALESCE($7, stowage_plan),
			hazardous = COALESCE($8, hazardous),
			notes = COALESCE($9, notes),
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		load.ID,
		nullableString(load.LoadPort),
		nullableString(load.DischargePort),
		nullableString(load.Commodity),
		nullableFloat(load.Quantity),
		nullableString(load.Unit),
		nullableBytes(load.StowagePlan),
		nullableBool(load.Hazardous),
		nullableString(load.Notes),
	).Scan(&load.UpdatedAt)
}

// Delete removes a cargo load.
func (repo *CargoLoadRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM shipman.cargo_loads WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id)
	return err
}
