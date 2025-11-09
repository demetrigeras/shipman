-- Vessels CRUD -------------------------------------------------------------

-- name: CreateVessel :one
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
RETURNING *;

-- name: GetVessel :one
SELECT *
FROM shipman.vessels
WHERE id = $1;

-- name: ListVessels :many
SELECT *
FROM shipman.vessels
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateVessel :one
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
RETURNING *;

-- name: DeleteVessel :exec
DELETE FROM shipman.vessels
WHERE id = $1;

