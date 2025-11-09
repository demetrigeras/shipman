-- Voyage Ports CRUD -------------------------------------------------------

-- name: CreateVoyagePort :one
INSERT INTO shipman.voyage_ports (
    voyage_id,
    port_name,
    port_country,
    port_unlocode,
    latitude,
    longitude,
    arrived_at,
    departed_at,
    laytime_hours,
    cargo_operations,
    notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: ListVoyagePorts :many
SELECT *
FROM shipman.voyage_ports
WHERE voyage_id = $1
ORDER BY arrived_at NULLS LAST, created_at;

-- name: UpdateVoyagePort :one
UPDATE shipman.voyage_ports
SET
    arrived_at = COALESCE($2, arrived_at),
    departed_at = COALESCE($3, departed_at),
    laytime_hours = COALESCE($4, laytime_hours),
    cargo_operations = COALESCE($5, cargo_operations),
    notes = COALESCE($6, notes),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteVoyagePort :exec
DELETE FROM shipman.voyage_ports
WHERE id = $1;

