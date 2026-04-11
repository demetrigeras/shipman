package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type DealVesselDetails struct {
	ID                 uuid.UUID  `json:"id"`
	DealID             uuid.UUID  `json:"deal_id"`
	FilledBy           uuid.UUID  `json:"filled_by"`
	VesselName         *string    `json:"vessel_name,omitempty"`
	IMONumber          *string    `json:"imo_number,omitempty"`
	VesselType         *string    `json:"vessel_type,omitempty"`
	FlagState          *string    `json:"flag_state,omitempty"`
	DeadweightTonnage  *float64   `json:"deadweight_tonnage,omitempty"`
	GrossTonnage       *float64   `json:"gross_tonnage,omitempty"`
	BuildYear          *int16     `json:"build_year,omitempty"`
	ClassSociety       *string    `json:"class_society,omitempty"`
	CurrentPosition    *string    `json:"current_position,omitempty"`
	AvailableFrom      *time.Time `json:"available_from,omitempty"`
	AskingRate         *float64   `json:"asking_rate,omitempty"`
	AskingRateCurrency string     `json:"asking_rate_currency"`
	AskingRateType     string     `json:"asking_rate_type"`
	Notes              *string    `json:"notes,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type DealCargoDetails struct {
	ID                  uuid.UUID  `json:"id"`
	DealID              uuid.UUID  `json:"deal_id"`
	FilledBy            uuid.UUID  `json:"filled_by"`
	Commodity           *string    `json:"commodity,omitempty"`
	Quantity            *float64   `json:"quantity,omitempty"`
	QuantityUnit        string     `json:"quantity_unit"`
	LoadPort            *string    `json:"load_port,omitempty"`
	DischargePort       *string    `json:"discharge_port,omitempty"`
	LaycanFrom          *time.Time `json:"laycan_from,omitempty"`
	LaycanTo            *time.Time `json:"laycan_to,omitempty"`
	FreightIdea         *float64   `json:"freight_idea,omitempty"`
	FreightCurrency     string     `json:"freight_currency"`
	FreightType         string     `json:"freight_type"`
	SpecialRequirements *string    `json:"special_requirements,omitempty"`
	Notes               *string    `json:"notes,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type DealDetailsRepository struct{}

func NewDealDetailsRepository() *DealDetailsRepository {
	return &DealDetailsRepository{}
}

func (r *DealDetailsRepository) UpsertVesselDetails(ctx context.Context, d *DealVesselDetails) error {
	const query = `
		INSERT INTO shipman.deal_vessel_details (
			deal_id, filled_by, vessel_name, imo_number, vessel_type, flag_state,
			deadweight_tonnage, gross_tonnage, build_year, class_society,
			current_position, available_from, asking_rate, asking_rate_currency,
			asking_rate_type, notes
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT (deal_id) DO UPDATE SET
			filled_by = EXCLUDED.filled_by,
			vessel_name = EXCLUDED.vessel_name,
			imo_number = EXCLUDED.imo_number,
			vessel_type = EXCLUDED.vessel_type,
			flag_state = EXCLUDED.flag_state,
			deadweight_tonnage = EXCLUDED.deadweight_tonnage,
			gross_tonnage = EXCLUDED.gross_tonnage,
			build_year = EXCLUDED.build_year,
			class_society = EXCLUDED.class_society,
			current_position = EXCLUDED.current_position,
			available_from = EXCLUDED.available_from,
			asking_rate = EXCLUDED.asking_rate,
			asking_rate_currency = EXCLUDED.asking_rate_currency,
			asking_rate_type = EXCLUDED.asking_rate_type,
			notes = EXCLUDED.notes,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	return Pool.QueryRowContext(ctx, query,
		d.DealID, d.FilledBy, d.VesselName, d.IMONumber, d.VesselType, d.FlagState,
		d.DeadweightTonnage, d.GrossTonnage, d.BuildYear, d.ClassSociety,
		d.CurrentPosition, d.AvailableFrom, d.AskingRate, d.AskingRateCurrency,
		d.AskingRateType, d.Notes,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (r *DealDetailsRepository) GetVesselDetails(ctx context.Context, dealID uuid.UUID) (*DealVesselDetails, error) {
	const query = `
		SELECT id, deal_id, filled_by, vessel_name, imo_number, vessel_type, flag_state,
			   deadweight_tonnage, gross_tonnage, build_year, class_society,
			   current_position, available_from, asking_rate, asking_rate_currency,
			   asking_rate_type, notes, created_at, updated_at
		FROM shipman.deal_vessel_details WHERE deal_id = $1
	`
	d := &DealVesselDetails{}
	var vesselName, imoNumber, vesselType, flagState, classSociety, currentPosition, notes sql.NullString
	var dwt, grt, askingRate sql.NullFloat64
	var buildYear sql.NullInt16
	var availableFrom sql.NullTime

	err := Pool.QueryRowContext(ctx, query, dealID).Scan(
		&d.ID, &d.DealID, &d.FilledBy, &vesselName, &imoNumber, &vesselType, &flagState,
		&dwt, &grt, &buildYear, &classSociety,
		&currentPosition, &availableFrom, &askingRate, &d.AskingRateCurrency,
		&d.AskingRateType, &notes, &d.CreatedAt, &d.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if vesselName.Valid { d.VesselName = &vesselName.String }
	if imoNumber.Valid { d.IMONumber = &imoNumber.String }
	if vesselType.Valid { d.VesselType = &vesselType.String }
	if flagState.Valid { d.FlagState = &flagState.String }
	if dwt.Valid { d.DeadweightTonnage = &dwt.Float64 }
	if grt.Valid { d.GrossTonnage = &grt.Float64 }
	if buildYear.Valid { d.BuildYear = &buildYear.Int16 }
	if classSociety.Valid { d.ClassSociety = &classSociety.String }
	if currentPosition.Valid { d.CurrentPosition = &currentPosition.String }
	if availableFrom.Valid { d.AvailableFrom = &availableFrom.Time }
	if askingRate.Valid { d.AskingRate = &askingRate.Float64 }
	if notes.Valid { d.Notes = &notes.String }

	return d, nil
}

func (r *DealDetailsRepository) UpsertCargoDetails(ctx context.Context, d *DealCargoDetails) error {
	const query = `
		INSERT INTO shipman.deal_cargo_details (
			deal_id, filled_by, commodity, quantity, quantity_unit,
			load_port, discharge_port, laycan_from, laycan_to,
			freight_idea, freight_currency, freight_type, special_requirements, notes
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (deal_id) DO UPDATE SET
			filled_by = EXCLUDED.filled_by,
			commodity = EXCLUDED.commodity,
			quantity = EXCLUDED.quantity,
			quantity_unit = EXCLUDED.quantity_unit,
			load_port = EXCLUDED.load_port,
			discharge_port = EXCLUDED.discharge_port,
			laycan_from = EXCLUDED.laycan_from,
			laycan_to = EXCLUDED.laycan_to,
			freight_idea = EXCLUDED.freight_idea,
			freight_currency = EXCLUDED.freight_currency,
			freight_type = EXCLUDED.freight_type,
			special_requirements = EXCLUDED.special_requirements,
			notes = EXCLUDED.notes,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	return Pool.QueryRowContext(ctx, query,
		d.DealID, d.FilledBy, d.Commodity, d.Quantity, d.QuantityUnit,
		d.LoadPort, d.DischargePort, d.LaycanFrom, d.LaycanTo,
		d.FreightIdea, d.FreightCurrency, d.FreightType, d.SpecialRequirements, d.Notes,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (r *DealDetailsRepository) GetCargoDetails(ctx context.Context, dealID uuid.UUID) (*DealCargoDetails, error) {
	const query = `
		SELECT id, deal_id, filled_by, commodity, quantity, quantity_unit,
			   load_port, discharge_port, laycan_from, laycan_to,
			   freight_idea, freight_currency, freight_type, special_requirements, notes,
			   created_at, updated_at
		FROM shipman.deal_cargo_details WHERE deal_id = $1
	`
	d := &DealCargoDetails{}
	var commodity, loadPort, dischargePort, specialReqs, notes sql.NullString
	var quantity, freightIdea sql.NullFloat64
	var laycanFrom, laycanTo sql.NullTime

	err := Pool.QueryRowContext(ctx, query, dealID).Scan(
		&d.ID, &d.DealID, &d.FilledBy, &commodity, &quantity, &d.QuantityUnit,
		&loadPort, &dischargePort, &laycanFrom, &laycanTo,
		&freightIdea, &d.FreightCurrency, &d.FreightType, &specialReqs, &notes,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if commodity.Valid { d.Commodity = &commodity.String }
	if quantity.Valid { d.Quantity = &quantity.Float64 }
	if loadPort.Valid { d.LoadPort = &loadPort.String }
	if dischargePort.Valid { d.DischargePort = &dischargePort.String }
	if laycanFrom.Valid { d.LaycanFrom = &laycanFrom.Time }
	if laycanTo.Valid { d.LaycanTo = &laycanTo.Time }
	if freightIdea.Valid { d.FreightIdea = &freightIdea.Float64 }
	if specialReqs.Valid { d.SpecialRequirements = &specialReqs.String }
	if notes.Valid { d.Notes = &notes.String }

	return d, nil
}
