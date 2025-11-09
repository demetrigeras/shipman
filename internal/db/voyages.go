package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Voyage mirrors shipman.voyages.
type Voyage struct {
	ID               uuid.UUID  `json:"id"`
	CharterDetailID  uuid.UUID  `json:"charter_detail_id"`
	VoyageNumber     *string    `json:"voyage_number,omitempty"`
	VesselName       *string    `json:"vessel_name,omitempty"`
	DeparturePort    *string    `json:"departure_port,omitempty"`
	ArrivalPort      *string    `json:"arrival_port,omitempty"`
	PlannedDeparture *time.Time `json:"planned_departure_at,omitempty"`
	PlannedArrival   *time.Time `json:"planned_arrival_at,omitempty"`
	ActualDeparture  *time.Time `json:"actual_departure_at,omitempty"`
	ActualArrival    *time.Time `json:"actual_arrival_at,omitempty"`
	DistanceNM       *float64   `json:"distance_nm,omitempty"`
	TimeAtSeaHours   *float64   `json:"time_at_sea_hours,omitempty"`
	FuelConsumedMT   *float64   `json:"fuel_consumed_mt,omitempty"`
	FuelType         *string    `json:"fuel_type,omitempty"`
	WeatherSummary   *string    `json:"weather_summary,omitempty"`
	Status           string     `json:"status"`
	Notes            *string    `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// VoyageService exposes CRUD behaviour.
type VoyageService interface {
	Create(ctx context.Context, v *Voyage) error
	Retrieve(ctx context.Context, id uuid.UUID) (Voyage, error)
	ListByCharter(ctx context.Context, charterID uuid.UUID) ([]Voyage, error)
	Update(ctx context.Context, v *Voyage) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// VoyageRepository implements VoyageService using Pool.
type VoyageRepository struct{}

// NewVoyageRepository returns repository.
func NewVoyageRepository() *VoyageRepository {
	return &VoyageRepository{}
}

// Create inserts voyage row.
func (repo *VoyageRepository) Create(ctx context.Context, v *Voyage) error {
	const query = `
		INSERT INTO shipman.voyages (
			charter_detail_id,
			voyage_number,
			vessel_name,
			departure_port,
			arrival_port,
			planned_departure_at,
			planned_arrival_at,
			actual_departure_at,
			actual_arrival_at,
			distance_nm,
			time_at_sea_hours,
			fuel_consumed_mt,
			fuel_type,
			weather_summary,
			status,
			notes
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14,
			$15,
			$16
		)
		RETURNING id, status, created_at, updated_at
	`

	status := v.Status
	if status == "" {
		status = "planned"
	}

	return Pool.QueryRowContext(
		ctx,
		query,
		v.CharterDetailID,
		nullableString(v.VoyageNumber),
		nullableString(v.VesselName),
		nullableString(v.DeparturePort),
		nullableString(v.ArrivalPort),
		nullableTime(v.PlannedDeparture),
		nullableTime(v.PlannedArrival),
		nullableTime(v.ActualDeparture),
		nullableTime(v.ActualArrival),
		nullableFloat(v.DistanceNM),
		nullableFloat(v.TimeAtSeaHours),
		nullableFloat(v.FuelConsumedMT),
		nullableString(v.FuelType),
		nullableString(v.WeatherSummary),
		status,
		nullableString(v.Notes),
	).Scan(&v.ID, &v.Status, &v.CreatedAt, &v.UpdatedAt)
}

// Retrieve fetches voyage by id.
func (repo *VoyageRepository) Retrieve(ctx context.Context, id uuid.UUID) (Voyage, error) {
	const query = `
		SELECT
			id,
			charter_detail_id,
			voyage_number,
			vessel_name,
			departure_port,
			arrival_port,
			planned_departure_at,
			planned_arrival_at,
			actual_departure_at,
			actual_arrival_at,
			distance_nm,
			time_at_sea_hours,
			fuel_consumed_mt,
			fuel_type,
			weather_summary,
			status,
			notes,
			created_at,
			updated_at
		FROM shipman.voyages
		WHERE id = $1
	`

	var (
		voyage       Voyage
		vNumber      sql.NullString
		vVessel      sql.NullString
		depart       sql.NullString
		arrive       sql.NullString
		etaLoad      sql.NullTime
		etaDischarge sql.NullTime
		actDepart    sql.NullTime
		actArrive    sql.NullTime
		dist         sql.NullFloat64
		timeSea      sql.NullFloat64
		fuelAmt      sql.NullFloat64
		fuelType     sql.NullString
		weather      sql.NullString
		status       sql.NullString
		notes        sql.NullString
	)

	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&voyage.ID,
		&voyage.CharterDetailID,
		&vNumber,
		&vVessel,
		&depart,
		&arrive,
		&etaLoad,
		&etaDischarge,
		&actDepart,
		&actArrive,
		&dist,
		&timeSea,
		&fuelAmt,
		&fuelType,
		&weather,
		&status,
		&notes,
		&voyage.CreatedAt,
		&voyage.UpdatedAt,
	)
	if err != nil {
		return Voyage{}, err
	}

	voyage.VoyageNumber = stringPtr(vNumber)
	voyage.VesselName = stringPtr(vVessel)
	voyage.DeparturePort = stringPtr(depart)
	voyage.ArrivalPort = stringPtr(arrive)
	voyage.PlannedDeparture = timePtr(etaLoad)
	voyage.PlannedArrival = timePtr(etaDischarge)
	voyage.ActualDeparture = timePtr(actDepart)
	voyage.ActualArrival = timePtr(actArrive)
	voyage.DistanceNM = floatPtr(dist)
	voyage.TimeAtSeaHours = floatPtr(timeSea)
	voyage.FuelConsumedMT = floatPtr(fuelAmt)
	voyage.FuelType = stringPtr(fuelType)
	voyage.WeatherSummary = stringPtr(weather)
	voyage.Status = defaultString(status, "planned")
	voyage.Notes = stringPtr(notes)

	return voyage, nil
}

// ListByCharter returns voyages belonging to a charter.
func (repo *VoyageRepository) ListByCharter(ctx context.Context, charterID uuid.UUID) ([]Voyage, error) {
	const query = `
		SELECT id, charter_detail_id, voyage_number, status, planned_departure_at, planned_arrival_at, created_at, updated_at
		FROM shipman.voyages
		WHERE charter_detail_id = $1
		ORDER BY planned_departure_at NULLS LAST, created_at DESC
	`

	rows, err := Pool.QueryContext(ctx, query, charterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var voyages []Voyage
	for rows.Next() {
		var (
			voyage  Voyage
			vNumber sql.NullString
			status  sql.NullString
			planDep sql.NullTime
			planArr sql.NullTime
		)
		if err := rows.Scan(
			&voyage.ID,
			&voyage.CharterDetailID,
			&vNumber,
			&status,
			&planDep,
			&planArr,
			&voyage.CreatedAt,
			&voyage.UpdatedAt,
		); err != nil {
			return nil, err
		}
		voyage.VoyageNumber = stringPtr(vNumber)
		voyage.Status = defaultString(status, "planned")
		voyage.PlannedDeparture = timePtr(planDep)
		voyage.PlannedArrival = timePtr(planArr)
		voyages = append(voyages, voyage)
	}
	return voyages, rows.Err()
}

// Update modifies voyage row.
func (repo *VoyageRepository) Update(ctx context.Context, v *Voyage) error {
	const query = `
		UPDATE shipman.voyages
		SET
			voyage_number = $2,
			vessel_name = $3,
			departure_port = $4,
			arrival_port = $5,
			eta_load_port = $6,
			eta_discharge_port = $7,
			actual_departure_at = $8,
			actual_arrival_at = $9,
			distance_nm = $10,
			time_at_sea_hours = $11,
			fuel_consumed_mt = $12,
			fuel_type = $13,
			weather_summary = $14,
			status = $15,
			notes = $16,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		v.ID,
		nullableString(v.VoyageNumber),
		nullableString(v.VesselName),
		nullableString(v.DeparturePort),
		nullableString(v.ArrivalPort),
		nullableTime(v.PlannedDeparture),
		nullableTime(v.PlannedArrival),
		nullableTime(v.ActualDeparture),
		nullableTime(v.ActualArrival),
		nullableFloat(v.DistanceNM),
		nullableFloat(v.TimeAtSeaHours),
		nullableFloat(v.FuelConsumedMT),
		nullableString(v.FuelType),
		nullableString(v.WeatherSummary),
		v.Status,
		nullableString(v.Notes),
	).Scan(&v.UpdatedAt)
}

// Delete removes a voyage.
func (repo *VoyageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM shipman.voyages WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id)
	return err
}
