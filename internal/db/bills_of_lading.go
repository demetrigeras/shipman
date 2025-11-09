package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// BillOfLading mirrors shipman.bills_of_lading rows.
type BillOfLading struct {
	ID               uuid.UUID  `json:"id"`
	CharterDetailID  uuid.UUID  `json:"charter_detail_id"`
	VoyageID         *uuid.UUID `json:"voyage_id,omitempty"`
	DocumentNumber   string     `json:"document_number"`
	IssueDate        *time.Time `json:"issue_date,omitempty"`
	Issuer           *string    `json:"issuer,omitempty"`
	Consignee        *string    `json:"consignee,omitempty"`
	NotifyParty      *string    `json:"notify_party,omitempty"`
	CargoDescription *string    `json:"cargo_description,omitempty"`
	Quantity         *float64   `json:"quantity,omitempty"`
	QuantityUnit     *string    `json:"quantity_unit,omitempty"`
	StorageURI       *string    `json:"storage_uri,omitempty"`
	Checksum         *string    `json:"checksum,omitempty"`
	EncryptedKey     []byte     `json:"encrypted_key,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// BillOfLadingService exposes CRUD behaviour.
type BillOfLadingService interface {
	Create(ctx context.Context, bl *BillOfLading) error
	Retrieve(ctx context.Context, id uuid.UUID) (BillOfLading, error)
	ListByCharter(ctx context.Context, charterID uuid.UUID) ([]BillOfLading, error)
	Update(ctx context.Context, bl *BillOfLading) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// BillOfLadingRepository implements BillOfLadingService using Pool.
type BillOfLadingRepository struct{}

// NewBillOfLadingRepository returns repo.
func NewBillOfLadingRepository() *BillOfLadingRepository {
	return &BillOfLadingRepository{}
}

// Create inserts a bill of lading.
func (repo *BillOfLadingRepository) Create(ctx context.Context, bl *BillOfLading) error {
	const query = `
		INSERT INTO shipman.bills_of_lading (
			charter_detail_id,
			voyage_id,
			document_number,
			issue_date,
			issuer,
			consignee,
			notify_party,
			cargo_description,
			quantity,
			quantity_unit,
			storage_uri,
			checksum,
			encrypted_key,
			notes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		RETURNING id, created_at, updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		bl.CharterDetailID,
		nullableUUID(bl.VoyageID),
		bl.DocumentNumber,
		nullableTime(bl.IssueDate),
		nullableString(bl.Issuer),
		nullableString(bl.Consignee),
		nullableString(bl.NotifyParty),
		nullableString(bl.CargoDescription),
		nullableFloat(bl.Quantity),
		nullableString(bl.QuantityUnit),
		nullableString(bl.StorageURI),
		nullableString(bl.Checksum),
		nullableBytes(bl.EncryptedKey),
		nullableString(bl.Notes),
	).Scan(&bl.ID, &bl.CreatedAt, &bl.UpdatedAt)
}

// Retrieve fetches a bill of lading by id.
func (repo *BillOfLadingRepository) Retrieve(ctx context.Context, id uuid.UUID) (BillOfLading, error) {
	const query = `
		SELECT
			id,
			charter_detail_id,
			voyage_id,
			document_number,
			issue_date,
			issuer,
			consignee,
			notify_party,
			cargo_description,
			quantity,
			quantity_unit,
			storage_uri,
			checksum,
			encrypted_key,
			notes,
			created_at,
			updated_at
		FROM shipman.bills_of_lading
		WHERE id = $1
	`

	var (
		bl        BillOfLading
		voyage    sql.NullString
		issueDate sql.NullTime
		issuer    sql.NullString
		consignee sql.NullString
		notify    sql.NullString
		cargo     sql.NullString
		quantity  sql.NullFloat64
		unit      sql.NullString
		storage   sql.NullString
		checksum  sql.NullString
		keyBytes  []byte
		notes     sql.NullString
	)

	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&bl.ID,
		&bl.CharterDetailID,
		&voyage,
		&bl.DocumentNumber,
		&issueDate,
		&issuer,
		&consignee,
		&notify,
		&cargo,
		&quantity,
		&unit,
		&storage,
		&checksum,
		&keyBytes,
		&notes,
		&bl.CreatedAt,
		&bl.UpdatedAt,
	)
	if err != nil {
		return BillOfLading{}, err
	}

	bl.VoyageID = uuidPtrNullable(voyage)
	bl.IssueDate = timePtr(issueDate)
	bl.Issuer = stringPtr(issuer)
	bl.Consignee = stringPtr(consignee)
	bl.NotifyParty = stringPtr(notify)
	bl.CargoDescription = stringPtr(cargo)
	bl.Quantity = floatPtr(quantity)
	bl.QuantityUnit = stringPtr(unit)
	bl.StorageURI = stringPtr(storage)
	bl.Checksum = stringPtr(checksum)
	bl.EncryptedKey = bytesOrNil(keyBytes)
	bl.Notes = stringPtr(notes)

	return bl, nil
}

// ListByCharter returns bills for a charter.
func (repo *BillOfLadingRepository) ListByCharter(ctx context.Context, charterID uuid.UUID) ([]BillOfLading, error) {
	const query = `
		SELECT id, charter_detail_id, document_number, issue_date, created_at, updated_at
		FROM shipman.bills_of_lading
		WHERE charter_detail_id = $1
		ORDER BY issue_date NULLS LAST, created_at DESC
	`

	rows, err := Pool.QueryContext(ctx, query, charterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bills []BillOfLading
	for rows.Next() {
		var (
			bl    BillOfLading
			issue sql.NullTime
		)
		if err := rows.Scan(
			&bl.ID,
			&bl.CharterDetailID,
			&bl.DocumentNumber,
			&issue,
			&bl.CreatedAt,
			&bl.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bl.IssueDate = timePtr(issue)
		bills = append(bills, bl)
	}
	return bills, rows.Err()
}

// Update modifies bill of lading fields.
func (repo *BillOfLadingRepository) Update(ctx context.Context, bl *BillOfLading) error {
	const query = `
		UPDATE shipman.bills_of_lading
		SET
			voyage_id = $2,
			document_number = $3,
			issue_date = $4,
			issuer = $5,
			consignee = $6,
			notify_party = $7,
			cargo_description = $8,
			quantity = $9,
			quantity_unit = $10,
			storage_uri = $11,
			checksum = $12,
			encrypted_key = $13,
			notes = $14,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	return Pool.QueryRowContext(
		ctx,
		query,
		bl.ID,
		nullableUUID(bl.VoyageID),
		bl.DocumentNumber,
		nullableTime(bl.IssueDate),
		nullableString(bl.Issuer),
		nullableString(bl.Consignee),
		nullableString(bl.NotifyParty),
		nullableString(bl.CargoDescription),
		nullableFloat(bl.Quantity),
		nullableString(bl.QuantityUnit),
		nullableString(bl.StorageURI),
		nullableString(bl.Checksum),
		nullableBytes(bl.EncryptedKey),
		nullableString(bl.Notes),
	).Scan(&bl.UpdatedAt)
}

// Delete removes a bill of lading.
func (repo *BillOfLadingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM shipman.bills_of_lading WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id)
	return err
}
