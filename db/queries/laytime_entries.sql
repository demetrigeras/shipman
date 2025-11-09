-- Laytime Entries CRUD ----------------------------------------------------

-- name: CreateLaytimeEntry :one
INSERT INTO shipman.laytime_entries (
    charter_detail_id,
    voyage_id,
    port_name,
    activity,
    started_at,
    ended_at,
    hours_counted,
    remarks
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: ListLaytimeEntriesForCharter :many
SELECT *
FROM shipman.laytime_entries
WHERE charter_detail_id = $1
ORDER BY started_at;

-- name: ListLaytimeEntriesForVoyage :many
SELECT *
FROM shipman.laytime_entries
WHERE voyage_id = $1
ORDER BY started_at;

-- name: UpdateLaytimeEntry :one
UPDATE shipman.laytime_entries
SET
    ended_at = COALESCE($2, ended_at),
    hours_counted = COALESCE($3, hours_counted),
    remarks = COALESCE($4, remarks),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteLaytimeEntry :exec
DELETE FROM shipman.laytime_entries
WHERE id = $1;

