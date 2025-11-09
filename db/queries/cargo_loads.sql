-- Cargo Loads CRUD ---------------------------------------------------------

-- name: CreateCargoLoad :one
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
RETURNING *;

-- name: GetCargoLoad :one
SELECT *
FROM shipman.cargo_loads
WHERE id = $1;

-- name: ListCargoLoadsForVoyage :many
SELECT *
FROM shipman.cargo_loads
WHERE voyage_id = $1
ORDER BY created_at DESC;

-- name: UpdateCargoLoad :one
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
RETURNING *;

-- name: DeleteCargoLoad :exec
DELETE FROM shipman.cargo_loads
WHERE id = $1;

