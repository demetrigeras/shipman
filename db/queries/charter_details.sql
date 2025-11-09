-- Charter Details CRUD ----------------------------------------------------

-- name: CreateCharterDetail :one
INSERT INTO shipman.charter_details (
    created_by_user_id,
    title,
    charter_reference_code,
    vessel_name,
    counterparty_name,
    status,
    start_date,
    end_date,
    laytime_allowance_hours,
    demurrage_rate,
    demurrage_currency,
    fuel_clause,
    payment_terms,
    ai_status,
    ai_document_path,
    ai_extracted_terms,
    last_reviewed_at,
    notes
) VALUES (
    $1, $2, $3, $4, $5, COALESCE($6, 'draft'), $7, $8, $9, $10, $11,
    $12, $13, COALESCE($14, 'pending'), $15, $16, $17, $18
)
RETURNING *;

-- name: GetCharterDetail :one
SELECT *
FROM shipman.charter_details
WHERE id = $1;

-- name: ListCharterDetails :many
SELECT *
FROM shipman.charter_details
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateCharterDetailStatus :one
UPDATE shipman.charter_details
SET status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateCharterDetailAi :one
UPDATE shipman.charter_details
SET
    ai_status = COALESCE($2, ai_status),
    ai_document_path = COALESCE($3, ai_document_path),
    ai_extracted_terms = COALESCE($4, ai_extracted_terms),
    last_reviewed_at = COALESCE($5, last_reviewed_at),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCharterDetail :exec
DELETE FROM shipman.charter_details
WHERE id = $1;

