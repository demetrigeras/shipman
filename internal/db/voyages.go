package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// Voyage mirrors shipman.voyages.
type Voyage struct {
	ID                  uuid.UUID  `json:"id"`
	CharterDetailID     *uuid.UUID `json:"charter_detail_id,omitempty"`
	DealID              *uuid.UUID `json:"deal_id,omitempty"`
	OwnerUserID         *uuid.UUID `json:"owner_user_id,omitempty"`
	VoyageNumber        *string    `json:"voyage_number,omitempty"`
	VesselName          *string    `json:"vessel_name,omitempty"`
	IMONumber           *string    `json:"imo_number,omitempty"`
	VesselType          *string    `json:"vessel_type,omitempty"`
	DWT                 *float64   `json:"dwt,omitempty"`
	FlagState           *string    `json:"flag_state,omitempty"`
	DeparturePort       *string    `json:"departure_port,omitempty"`
	ArrivalPort         *string    `json:"arrival_port,omitempty"`
	PlannedDeparture    *time.Time `json:"planned_departure_at,omitempty"`
	PlannedArrival      *time.Time `json:"planned_arrival_at,omitempty"`
	ActualDeparture     *time.Time `json:"actual_departure_at,omitempty"`
	ActualArrival       *time.Time `json:"actual_arrival_at,omitempty"`
	DistanceNM          *float64   `json:"distance_nm,omitempty"`
	TimeAtSeaHours      *float64   `json:"time_at_sea_hours,omitempty"`
	FuelConsumedMT      *float64   `json:"fuel_consumed_mt,omitempty"`
	FuelType            *string    `json:"fuel_type,omitempty"`
	WeatherSummary      *string    `json:"weather_summary,omitempty"`
	// Commercial terms
	HireRate            *float64   `json:"hire_rate,omitempty"`
	FreightRate         *float64   `json:"freight_rate,omitempty"`
	CargoQuantity       *float64   `json:"cargo_quantity,omitempty"`
	CargoType           *string    `json:"cargo_type,omitempty"`
	// Laytime / demurrage terms
	LaytimeAllowedHours *float64   `json:"laytime_allowed_hours,omitempty"`
	DemurrageRate       *float64   `json:"demurrage_rate,omitempty"`
	DespatchRate        *float64   `json:"despatch_rate,omitempty"`
	DemurrageCurrency   string     `json:"demurrage_currency"`
	// Payment schedule terms
	PaymentFrequency    *string    `json:"payment_frequency,omitempty"`
	FirstPaymentDate    *time.Time `json:"first_payment_date,omitempty"`
	TotalContractValue  *float64   `json:"total_contract_value,omitempty"`
	CommissionRate      *float64   `json:"commission_rate,omitempty"`
	BunkerCost          *float64   `json:"bunker_cost,omitempty"`
	PortCosts           *float64   `json:"port_costs,omitempty"`
	InsuranceCost       *float64   `json:"insurance_cost,omitempty"`
	CounterpartyName    *string    `json:"counterparty_name,omitempty"`
	CounterpartyEmail   *string    `json:"counterparty_email,omitempty"`
	// Linked users for the two non-owner parties. Set when somebody accepts
	// an invite; gives the FE owner/counterparty/broker access checks and
	// makes voyages appear in the joined user's `/voyages` list.
	CounterpartyUserID  *uuid.UUID `json:"counterparty_user_id,omitempty"`
	BrokerUserID        *uuid.UUID `json:"broker_user_id,omitempty"`
	DocumentID          *uuid.UUID `json:"document_id,omitempty"`
	CharterType         *string    `json:"charter_type,omitempty"`
	Status              string     `json:"status"`
	Notes               *string    `json:"notes,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// LaytimeSummary is computed from laytime_entries for a voyage.
type LaytimeSummary struct {
	TotalHoursUsed    float64  `json:"total_hours_used"`
	TotalHoursAllowed float64  `json:"total_hours_allowed"`
	BalanceHours      float64  `json:"balance_hours"`      // negative = demurrage
	DemurrageHours    float64  `json:"demurrage_hours"`
	DespatchHours     float64  `json:"despatch_hours"`
	DemurrageAmount   *float64 `json:"demurrage_amount,omitempty"`
	DespatchAmount    *float64 `json:"despatch_amount,omitempty"`
	Currency          string   `json:"currency"`
}

// VoyageRepository implements voyage database access.
type VoyageRepository struct{}

func NewVoyageRepository() *VoyageRepository {
	return &VoyageRepository{}
}

func (repo *VoyageRepository) Create(ctx context.Context, v *Voyage) error {
	const query = `
		INSERT INTO shipman.voyages (
			charter_detail_id, deal_id, owner_user_id,
			voyage_number, vessel_name, imo_number, vessel_type, dwt, flag_state,
			departure_port, arrival_port,
			planned_departure_at, planned_arrival_at,
			hire_rate, freight_rate, cargo_quantity, cargo_type,
			laytime_allowed_hours, demurrage_rate, despatch_rate, demurrage_currency,
			payment_frequency, first_payment_date, total_contract_value,
			commission_rate, bunker_cost, port_costs, insurance_cost,
			counterparty_name, counterparty_email,
			charter_type, status, notes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18, $19, $20,
			COALESCE($21, 'USD'),
			$22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, COALESCE($32, 'planned'), $33
		)
		RETURNING id, status, demurrage_currency, created_at, updated_at
	`
	return Pool.QueryRowContext(ctx, query,
		nullableUUID(v.CharterDetailID),
		nullableUUID(v.DealID),
		nullableUUID(v.OwnerUserID),
		nullableString(v.VoyageNumber),
		nullableString(v.VesselName),
		nullableString(v.IMONumber),
		nullableString(v.VesselType),
		nullableFloat(v.DWT),
		nullableString(v.FlagState),
		nullableString(v.DeparturePort),
		nullableString(v.ArrivalPort),
		nullableTime(v.PlannedDeparture),
		nullableTime(v.PlannedArrival),
		nullableFloat(v.HireRate),
		nullableFloat(v.FreightRate),
		nullableFloat(v.CargoQuantity),
		nullableString(v.CargoType),
		nullableFloat(v.LaytimeAllowedHours),
		nullableFloat(v.DemurrageRate),
		nullableFloat(v.DespatchRate),
		nullableString(&v.DemurrageCurrency),
		nullableString(v.PaymentFrequency),
		nullableTime(v.FirstPaymentDate),
		nullableFloat(v.TotalContractValue),
		nullableFloat(v.CommissionRate),
		nullableFloat(v.BunkerCost),
		nullableFloat(v.PortCosts),
		nullableFloat(v.InsuranceCost),
		nullableString(v.CounterpartyName),
		nullableString(v.CounterpartyEmail),
		nullableString(v.CharterType),
		nullableString(&v.Status),
		nullableString(v.Notes),
	).Scan(&v.ID, &v.Status, &v.DemurrageCurrency, &v.CreatedAt, &v.UpdatedAt)
}

func (repo *VoyageRepository) AttachDocument(ctx context.Context, voyageID, documentID uuid.UUID) error {
	const query = `UPDATE shipman.voyages SET document_id = $2, updated_at = NOW() WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, voyageID, documentID)
	return err
}

func (repo *VoyageRepository) Retrieve(ctx context.Context, id uuid.UUID) (Voyage, error) {
	const query = `
		SELECT
			id, charter_detail_id, deal_id, owner_user_id,
			voyage_number, vessel_name, imo_number, vessel_type, dwt, flag_state,
			departure_port, arrival_port,
			planned_departure_at, planned_arrival_at,
			actual_departure_at, actual_arrival_at,
			distance_nm, time_at_sea_hours,
			fuel_consumed_mt, fuel_type, weather_summary,
			hire_rate, freight_rate, cargo_quantity, cargo_type,
			laytime_allowed_hours, demurrage_rate, despatch_rate,
			COALESCE(demurrage_currency, 'USD'),
			payment_frequency, first_payment_date, total_contract_value,
			commission_rate, bunker_cost, port_costs, insurance_cost,
			counterparty_name, counterparty_email,
			counterparty_user_id, broker_user_id,
			document_id, charter_type,
			status, notes, created_at, updated_at
		FROM shipman.voyages
		WHERE id = $1
	`
	var (
		v               Voyage
		charterID       sql.NullString
		dealID          sql.NullString
		ownerID         sql.NullString
		vNumber         sql.NullString
		vesselName      sql.NullString
		imo             sql.NullString
		vType           sql.NullString
		dwt             sql.NullFloat64
		flag            sql.NullString
		departPort      sql.NullString
		arrivePort      sql.NullString
		planDep         sql.NullTime
		planArr         sql.NullTime
		actDep          sql.NullTime
		actArr          sql.NullTime
		distNM          sql.NullFloat64
		timeSea         sql.NullFloat64
		fuelAmt         sql.NullFloat64
		fuelType        sql.NullString
		weather         sql.NullString
		hireRate        sql.NullFloat64
		freightRate     sql.NullFloat64
		cargoQty        sql.NullFloat64
		cargoType       sql.NullString
		laytimeHrs      sql.NullFloat64
		demRate         sql.NullFloat64
		despRate        sql.NullFloat64
		payFreq         sql.NullString
		firstPayDate    sql.NullTime
		totalValue      sql.NullFloat64
		commRate        sql.NullFloat64
		bunkerCost      sql.NullFloat64
		portCosts       sql.NullFloat64
		insuranceCost   sql.NullFloat64
		counterName     sql.NullString
		counterEmail    sql.NullString
		counterUserID   sql.NullString
		brokerUserID    sql.NullString
		documentID      sql.NullString
		charterType     sql.NullString
		notes           sql.NullString
	)
	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&v.ID, &charterID, &dealID, &ownerID,
		&vNumber, &vesselName, &imo, &vType, &dwt, &flag,
		&departPort, &arrivePort,
		&planDep, &planArr, &actDep, &actArr,
		&distNM, &timeSea, &fuelAmt, &fuelType, &weather,
		&hireRate, &freightRate, &cargoQty, &cargoType,
		&laytimeHrs, &demRate, &despRate,
		&v.DemurrageCurrency,
		&payFreq, &firstPayDate, &totalValue,
		&commRate, &bunkerCost, &portCosts, &insuranceCost,
		&counterName, &counterEmail,
		&counterUserID, &brokerUserID,
		&documentID, &charterType,
		&v.Status, &notes, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return Voyage{}, err
	}
	v.CharterDetailID = uuidPtrNullable(charterID)
	v.DealID = uuidPtrNullable(dealID)
	v.OwnerUserID = uuidPtrNullable(ownerID)
	v.VoyageNumber = stringPtr(vNumber)
	v.VesselName = stringPtr(vesselName)
	v.IMONumber = stringPtr(imo)
	v.VesselType = stringPtr(vType)
	v.DWT = floatPtr(dwt)
	v.FlagState = stringPtr(flag)
	v.DeparturePort = stringPtr(departPort)
	v.ArrivalPort = stringPtr(arrivePort)
	v.PlannedDeparture = timePtr(planDep)
	v.PlannedArrival = timePtr(planArr)
	v.ActualDeparture = timePtr(actDep)
	v.ActualArrival = timePtr(actArr)
	v.DistanceNM = floatPtr(distNM)
	v.TimeAtSeaHours = floatPtr(timeSea)
	v.FuelConsumedMT = floatPtr(fuelAmt)
	v.FuelType = stringPtr(fuelType)
	v.WeatherSummary = stringPtr(weather)
	v.HireRate = floatPtr(hireRate)
	v.FreightRate = floatPtr(freightRate)
	v.CargoQuantity = floatPtr(cargoQty)
	v.CargoType = stringPtr(cargoType)
	v.LaytimeAllowedHours = floatPtr(laytimeHrs)
	v.DemurrageRate = floatPtr(demRate)
	v.DespatchRate = floatPtr(despRate)
	v.PaymentFrequency = stringPtr(payFreq)
	v.FirstPaymentDate = timePtr(firstPayDate)
	v.TotalContractValue = floatPtr(totalValue)
	v.CommissionRate = floatPtr(commRate)
	v.BunkerCost = floatPtr(bunkerCost)
	v.PortCosts = floatPtr(portCosts)
	v.InsuranceCost = floatPtr(insuranceCost)
	v.CounterpartyName = stringPtr(counterName)
	v.CounterpartyEmail = stringPtr(counterEmail)
	v.CounterpartyUserID = uuidPtrNullable(counterUserID)
	v.BrokerUserID = uuidPtrNullable(brokerUserID)
	v.DocumentID = uuidPtrNullable(documentID)
	v.CharterType = stringPtr(charterType)
	v.Notes = stringPtr(notes)
	return v, nil
}

func (repo *VoyageRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]Voyage, error) {
	// Return every voyage the user is involved in — owner, counterparty
	// (the joined-via-invite side), or broker. Without this any invited user
	// would see an empty /voyages page after accepting.
	const query = `
		SELECT id, deal_id, voyage_number, vessel_name, imo_number,
		       departure_port, arrival_port,
		       planned_departure_at, planned_arrival_at,
		       actual_departure_at, actual_arrival_at,
		       cargo_type, cargo_quantity,
		       counterparty_user_id, broker_user_id, owner_user_id,
		       status, created_at, updated_at
		FROM shipman.voyages
		WHERE owner_user_id = $1
		   OR counterparty_user_id = $1
		   OR broker_user_id = $1
		ORDER BY COALESCE(planned_departure_at, created_at) DESC
	`
	rows, err := Pool.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var voyages []Voyage
	for rows.Next() {
		var (
			v             Voyage
			dealID        sql.NullString
			vNumber       sql.NullString
			vessel        sql.NullString
			imo           sql.NullString
			depPort       sql.NullString
			arrPort       sql.NullString
			planDep       sql.NullTime
			planArr       sql.NullTime
			actDep        sql.NullTime
			actArr        sql.NullTime
			cargoType     sql.NullString
			cargoQty      sql.NullFloat64
			counterUserID sql.NullString
			brokerUserID  sql.NullString
			ownerUserID   sql.NullString
		)
		if err := rows.Scan(
			&v.ID, &dealID, &vNumber, &vessel, &imo,
			&depPort, &arrPort,
			&planDep, &planArr, &actDep, &actArr,
			&cargoType, &cargoQty,
			&counterUserID, &brokerUserID, &ownerUserID,
			&v.Status, &v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, err
		}
		v.DealID = uuidPtrNullable(dealID)
		v.VoyageNumber = stringPtr(vNumber)
		v.VesselName = stringPtr(vessel)
		v.IMONumber = stringPtr(imo)
		v.DeparturePort = stringPtr(depPort)
		v.ArrivalPort = stringPtr(arrPort)
		v.PlannedDeparture = timePtr(planDep)
		v.PlannedArrival = timePtr(planArr)
		v.ActualDeparture = timePtr(actDep)
		v.ActualArrival = timePtr(actArr)
		v.CargoType = stringPtr(cargoType)
		v.CargoQuantity = floatPtr(cargoQty)
		v.CounterpartyUserID = uuidPtrNullable(counterUserID)
		v.BrokerUserID = uuidPtrNullable(brokerUserID)
		v.OwnerUserID = uuidPtrNullable(ownerUserID)
		voyages = append(voyages, v)
	}
	return voyages, rows.Err()
}

// IsParticipant returns true when the user is owner, counterparty, or broker
// on the voyage. Used by all read/write access checks in the voyage handlers.
func (repo *VoyageRepository) IsParticipant(ctx context.Context, voyageID, userID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM shipman.voyages
			WHERE id = $1
			  AND (owner_user_id = $2 OR counterparty_user_id = $2 OR broker_user_id = $2)
		)
	`
	var exists bool
	err := Pool.QueryRowContext(ctx, query, voyageID, userID).Scan(&exists)
	return exists, err
}

// SetParty stamps user_id into the voyage's role column. role must be one of
// 'shipowner', 'charterer', 'broker'. The shipowner role implicitly maps to
// owner_user_id (which already exists), while charterer maps to
// counterparty_user_id. The column-name interpolation is safe because role
// is whitelisted.
func (repo *VoyageRepository) SetParty(ctx context.Context, voyageID uuid.UUID, role string, userID uuid.UUID) error {
	var col string
	switch role {
	case "broker":
		col = "broker_user_id"
	case "shipowner":
		col = "owner_user_id"
	case "charterer":
		col = "counterparty_user_id"
	default:
		// Fall back to counterparty for any unknown role so the user still
		// has access — better than silently dropping the link.
		col = "counterparty_user_id"
	}
	q := "UPDATE shipman.voyages SET " + col + " = $2 WHERE id = $1"
	_, err := Pool.ExecContext(ctx, q, voyageID, userID)
	return err
}

func (repo *VoyageRepository) Update(ctx context.Context, v *Voyage) error {
	const query = `
		UPDATE shipman.voyages
		SET
			voyage_number = $2, vessel_name = $3, imo_number = $4,
			vessel_type = $5, dwt = $6, flag_state = $7,
			departure_port = $8, arrival_port = $9,
			planned_departure_at = $10, planned_arrival_at = $11,
			actual_departure_at = $12, actual_arrival_at = $13,
			distance_nm = $14, time_at_sea_hours = $15,
			fuel_consumed_mt = $16, fuel_type = $17, weather_summary = $18,
			hire_rate = $19, freight_rate = $20,
			cargo_quantity = $21, cargo_type = $22,
			laytime_allowed_hours = $23, demurrage_rate = $24, despatch_rate = $25,
			demurrage_currency = $26,
			payment_frequency = $27, first_payment_date = $28,
			total_contract_value = $29, commission_rate = $30,
			bunker_cost = $31, port_costs = $32, insurance_cost = $33,
			counterparty_name = $34, counterparty_email = $35,
			status = $36, notes = $37,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	return Pool.QueryRowContext(ctx, query,
		v.ID,
		nullableString(v.VoyageNumber), nullableString(v.VesselName), nullableString(v.IMONumber),
		nullableString(v.VesselType), nullableFloat(v.DWT), nullableString(v.FlagState),
		nullableString(v.DeparturePort), nullableString(v.ArrivalPort),
		nullableTime(v.PlannedDeparture), nullableTime(v.PlannedArrival),
		nullableTime(v.ActualDeparture), nullableTime(v.ActualArrival),
		nullableFloat(v.DistanceNM), nullableFloat(v.TimeAtSeaHours),
		nullableFloat(v.FuelConsumedMT), nullableString(v.FuelType), nullableString(v.WeatherSummary),
		nullableFloat(v.HireRate), nullableFloat(v.FreightRate),
		nullableFloat(v.CargoQuantity), nullableString(v.CargoType),
		nullableFloat(v.LaytimeAllowedHours), nullableFloat(v.DemurrageRate), nullableFloat(v.DespatchRate),
		v.DemurrageCurrency,
		nullableString(v.PaymentFrequency), nullableTime(v.FirstPaymentDate),
		nullableFloat(v.TotalContractValue), nullableFloat(v.CommissionRate),
		nullableFloat(v.BunkerCost), nullableFloat(v.PortCosts), nullableFloat(v.InsuranceCost),
		nullableString(v.CounterpartyName), nullableString(v.CounterpartyEmail),
		v.Status, nullableString(v.Notes),
	).Scan(&v.UpdatedAt)
}

func (repo *VoyageRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	const query = `UPDATE shipman.voyages SET status = $2, updated_at = NOW() WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id, status)
	return err
}

func (repo *VoyageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := Pool.ExecContext(ctx, `DELETE FROM shipman.voyages WHERE id = $1`, id)
	return err
}

// CalcLaytime sums hours_counted from laytime_entries and computes demurrage/despatch.
func (repo *VoyageRepository) CalcLaytime(ctx context.Context, voyageID uuid.UUID) (LaytimeSummary, error) {
	// Sum countable hours (exclude records marked as excluded activities)
	const sumQuery = `
		SELECT COALESCE(SUM(hours_counted), 0)
		FROM shipman.laytime_entries
		WHERE voyage_id = $1
		  AND hours_counted IS NOT NULL
	`
	var totalUsed float64
	if err := Pool.QueryRowContext(ctx, sumQuery, voyageID).Scan(&totalUsed); err != nil {
		return LaytimeSummary{}, err
	}

	// Get voyage terms
	const termsQuery = `
		SELECT COALESCE(laytime_allowed_hours, 0),
		       COALESCE(demurrage_rate, 0),
		       COALESCE(despatch_rate, 0),
		       COALESCE(demurrage_currency, 'USD')
		FROM shipman.voyages WHERE id = $1
	`
	var allowed, demRate, despRate float64
	var currency string
	if err := Pool.QueryRowContext(ctx, termsQuery, voyageID).Scan(&allowed, &demRate, &despRate, &currency); err != nil {
		return LaytimeSummary{}, err
	}

	balance := allowed - totalUsed // positive = under = despatch; negative = over = demurrage
	summary := LaytimeSummary{
		TotalHoursUsed:    totalUsed,
		TotalHoursAllowed: allowed,
		BalanceHours:      balance,
		Currency:          currency,
	}

	if balance < 0 {
		// demurrage
		summary.DemurrageHours = -balance
		if demRate > 0 {
			amt := (summary.DemurrageHours / 24) * demRate
			summary.DemurrageAmount = &amt
		}
	} else if balance > 0 {
		// despatch
		summary.DespatchHours = balance
		if despRate > 0 {
			amt := (summary.DespatchHours / 24) * despRate
			summary.DespatchAmount = &amt
		}
	}

	return summary, nil
}

// ── Voyage Invites ────────────────────────────────────────────────────────────

type VoyageInvite struct {
	ID           uuid.UUID  `json:"id"`
	VoyageID     uuid.UUID  `json:"voyage_id"`
	Token        string     `json:"token"`
	Role         string     `json:"role"`
	InvitedEmail string     `json:"invited_email"`
	CreatedBy    uuid.UUID  `json:"created_by"`
	ExpiresAt    time.Time  `json:"expires_at"`
	UsedAt       *time.Time `json:"used_at,omitempty"`
	UsedBy       *uuid.UUID `json:"used_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (repo *VoyageRepository) CreateInvite(ctx context.Context, i *VoyageInvite) error {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	i.Token = hex.EncodeToString(b)

	const query = `
		INSERT INTO shipman.voyage_invites
			(voyage_id, token, role, invited_email, created_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	return Pool.QueryRowContext(ctx, query,
		i.VoyageID, i.Token, i.Role, i.InvitedEmail, i.CreatedBy, i.ExpiresAt,
	).Scan(&i.ID, &i.CreatedAt)
}

func (repo *VoyageRepository) GetInviteByToken(ctx context.Context, token string) (VoyageInvite, error) {
	const query = `
		SELECT id, voyage_id, token, role, COALESCE(invited_email, ''), created_by,
		       expires_at, used_at, used_by, created_at
		FROM shipman.voyage_invites
		WHERE token = $1
	`
	var i VoyageInvite
	var usedAt sql.NullTime
	var usedBy sql.NullString

	err := Pool.QueryRowContext(ctx, query, token).Scan(
		&i.ID, &i.VoyageID, &i.Token, &i.Role, &i.InvitedEmail, &i.CreatedBy,
		&i.ExpiresAt, &usedAt, &usedBy, &i.CreatedAt,
	)
	if err != nil {
		return i, err
	}
	if usedAt.Valid {
		i.UsedAt = &usedAt.Time
	}
	if usedBy.Valid {
		if uid, err2 := uuid.Parse(usedBy.String); err2 == nil {
			i.UsedBy = &uid
		}
	}
	return i, nil
}

func (repo *VoyageRepository) UseInvite(ctx context.Context, token string, usedBy uuid.UUID) error {
	const query = `
		UPDATE shipman.voyage_invites
		SET used_at = NOW(), used_by = $2
		WHERE token = $1
	`
	_, err := Pool.ExecContext(ctx, query, token, usedBy)
	return err
}
