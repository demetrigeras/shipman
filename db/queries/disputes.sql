-- Disputes CRUD -----------------------------------------------------------

-- name: CreateDispute :one
INSERT INTO shipman.disputes (
    charter_detail_id,
    voyage_id,
    payment_id,
    laytime_entry_id,
    raised_by_user_id,
    subject,
    description,
    claimed_amount,
    currency,
    status,
    resolution_notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, COALESCE($10, 'open'), $11
)
RETURNING *;

-- name: GetDispute :one
SELECT *
FROM shipman.disputes
WHERE id = $1;

-- name: ListDisputesForCharter :many
SELECT *
FROM shipman.disputes
WHERE charter_detail_id = $1
ORDER BY created_at DESC;

-- name: UpdateDisputeStatus :one
UPDATE shipman.disputes
SET
    status = COALESCE($2, status),
    resolution_notes = COALESCE($3, resolution_notes),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDispute :exec
DELETE FROM shipman.disputes
WHERE id = $1;

