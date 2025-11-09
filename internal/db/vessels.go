package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Vessel mirrors shipman.vessels rows.
type Vessel struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	IMONumber         *string   `json:"imo_number,omitempty"`
	FlagState         *string   `json:"flag_state,omitempty"`
	VesselType        *string   `json:"vessel_type,omitempty"`
	CallSign          *string   `json:"call_sign,omitempty"`
	DeadweightTonnage *float64  `json:"deadweight_tonnage,omitempty"`
	GrossTonnage      *float64  `json:"gross_tonnage,omitempty"`
	NetTonnage        *float64  `json:"net_tonnage,omitempty"`
	Capacity          []byte    `json:"capacity,omitempty"` // JSON blob
	BuildYear         *int16    `json:"build_year,omitempty"`
	ClassSociety      *string   `json:"class_society,omitempty"`
	Owner             *string   `json:"owner,omitempty"`
	Manager           *string   `json:"manager,omitempty"`
	DocumentationURI  *string   `json:"documentation_uri,omitempty"`
	Notes             *string   `json:"notes,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// VesselService exposes CRUD behaviour.
type VesselService interface {
	Create(ctx context.Context, vessel *Vessel) error
	Retrieve(ctx context.Context, id uuid.UUID) (Vessel, error)
	List(ctx context.Context, limit, offset int) ([]Vessel, error)
	Update(ctx context.Context, vessel *Vessel) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// VesselRepository implements VesselService using Pool.
type VesselRepository struct{}

// NewVesselRepository returns a repository.
func NewVesselRepository() *VesselRepository {
	return &VesselRepository{}
}

// Create inserts a vessel.
func (repo *VesselRepository) Create(ctx context.Context, vessel *Vessel) error {
	const query = `
		INSERT INTO shipman.vessels (
			name,
			imo_number,
			flag_state,
			vessel_type,
			call_sign,
			deadweight_tonnage,
			gross_tonnage,
			net_tonnage,
			capacity,
			build_year,
			class_society,
			owner,
			manager,
			documentation_uri,
			notes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		RETURNING id, created_at, updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		vessel.Name,
		nullableString(vessel.IMONumber),
		nullableString(vessel.FlagState),
		nullableString(vessel.VesselType),
		nullableString(vessel.CallSign),
		nullableFloat(vessel.DeadweightTonnage),
		nullableFloat(vessel.GrossTonnage),
		nullableFloat(vessel.NetTonnage),
		nullableBytes(vessel.Capacity),
		nullableInt16(vessel.BuildYear),
		nullableString(vessel.ClassSociety),
		nullableString(vessel.Owner),
		nullableString(vessel.Manager),
		nullableString(vessel.DocumentationURI),
		nullableString(vessel.Notes),
	).Scan(&vessel.ID, &vessel.CreatedAt, &vessel.UpdatedAt)
}

// Retrieve fetches a vessel by id.
func (repo *VesselRepository) Retrieve(ctx context.Context, id uuid.UUID) (Vessel, error) {
	const query = `
		SELECT
			id,
			name,
			imo_number,
			flag_state,
			vessel_type,
			call_sign,
			deadweight_tonnage,
			gross_tonnage,
			net_tonnage,
			capacity,
			build_year,
			class_society,
			owner,
			manager,
			documentation_uri,
			notes,
			created_at,
			updated_at
		FROM shipman.vessels
		WHERE id = $1
	`

	var (
		vessel    Vessel
		imo       sql.NullString
		flag      sql.NullString
		vType     sql.NullString
		callSign  sql.NullString
		dwt       sql.NullFloat64
		gross     sql.NullFloat64
		net       sql.NullFloat64
		capacity  []byte
		buildYear sql.NullInt16
		classSoc  sql.NullString
		owner     sql.NullString
		manager   sql.NullString
		docURI    sql.NullString
		notes     sql.NullString
	)

	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&vessel.ID,
		&vessel.Name,
		&imo,
		&flag,
		&vType,
		&callSign,
		&dwt,
		&gross,
		&net,
		&capacity,
		&buildYear,
		&classSoc,
		&owner,
		&manager,
		&docURI,
		&notes,
		&vessel.CreatedAt,
		&vessel.UpdatedAt,
	)
	if err != nil {
		return Vessel{}, err
	}

	vessel.IMONumber = stringPtr(imo)
	vessel.FlagState = stringPtr(flag)
	vessel.VesselType = stringPtr(vType)
	vessel.CallSign = stringPtr(callSign)
	vessel.DeadweightTonnage = floatPtr(dwt)
	vessel.GrossTonnage = floatPtr(gross)
	vessel.NetTonnage = floatPtr(net)
	vessel.Capacity = bytesOrNil(capacity)
	vessel.BuildYear = int16Ptr(buildYear)
	vessel.ClassSociety = stringPtr(classSoc)
	vessel.Owner = stringPtr(owner)
	vessel.Manager = stringPtr(manager)
	vessel.DocumentationURI = stringPtr(docURI)
	vessel.Notes = stringPtr(notes)

	return vessel, nil
}

// List returns vessels ordered by newest first.
func (repo *VesselRepository) List(ctx context.Context, limit, offset int) ([]Vessel, error) {
	const query = `
		SELECT id, name, imo_number, created_at, updated_at
		FROM shipman.vessels
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := Pool.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vessels []Vessel
	for rows.Next() {
		var (
			vessel Vessel
			imo    sql.NullString
		)
		if err := rows.Scan(
			&vessel.ID,
			&vessel.Name,
			&imo,
			&vessel.CreatedAt,
			&vessel.UpdatedAt,
		); err != nil {
			return nil, err
		}
		vessel.IMONumber = stringPtr(imo)
		vessels = append(vessels, vessel)
	}
	return vessels, rows.Err()
}

// Update modifies vessel fields.
func (repo *VesselRepository) Update(ctx context.Context, vessel *Vessel) error {
	const query = `
		UPDATE shipman.vessels
		SET
			name = $2,
			imo_number = $3,
			flag_state = $4,
			vessel_type = $5,
			call_sign = $6,
			deadweight_tonnage = $7,
			gross_tonnage = $8,
			net_tonnage = $9,
			capacity = $10,
			build_year = $11,
			class_society = $12,
			owner = $13,
			manager = $14,
			documentation_uri = $15,
			notes = $16,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		vessel.ID,
		vessel.Name,
		nullableString(vessel.IMONumber),
		nullableString(vessel.FlagState),
		nullableString(vessel.VesselType),
		nullableString(vessel.CallSign),
		nullableFloat(vessel.DeadweightTonnage),
		nullableFloat(vessel.GrossTonnage),
		nullableFloat(vessel.NetTonnage),
		nullableBytes(vessel.Capacity),
		nullableInt16(vessel.BuildYear),
		nullableString(vessel.ClassSociety),
		nullableString(vessel.Owner),
		nullableString(vessel.Manager),
		nullableString(vessel.DocumentationURI),
		nullableString(vessel.Notes),
	).Scan(&vessel.UpdatedAt)
}

// Delete removes a vessel.
func (repo *VesselRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM shipman.vessels WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id)
	return err
}
