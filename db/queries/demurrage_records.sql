-- Demurrage Records CRUD ---------------------------------------------------

-- name: CreateDemurrageRecord :one
INSERT INTO shipman.demurrage_records (
    charter_detail_id,
    voyage_id,
    laytime_entry_id,
    claimed_hours,
    claimed_amount,
    currency,
    status,
    reference,
    supporting_doc_uri,
    notes
) VALUES (
    $1, $2, $3, $4, $5, COALESCE($6, 'USD'), COALESCE($7, 'draft'), $8, $9, $10
)
RETURNING *;

-- name: GetDemurrageRecord :one
SELECT *
FROM shipman.demurrage_records
WHERE id = $1;

-- name: ListDemurrageRecordsForCharter :many
SELECT *
FROM shipman.demurrage_records
WHERE charter_detail_id = $1
ORDER BY created_at DESC;

-- name: UpdateDemurrageRecordStatus :one
UPDATE shipman.demurrage_records
SET
    status = COALESCE($2, status),
    claimed_hours = COALESCE($3, claimed_hours),
    claimed_amount = COALESCE($4, claimed_amount),
    currency = COALESCE($5, currency),
    reference = COALESCE($6, reference),
    supporting_doc_uri = COALESCE($7, supporting_doc_uri),
    notes = COALESCE($8, notes),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDemurrageRecord :exec
DELETE FROM shipman.demurrage_records
WHERE id = $1;

