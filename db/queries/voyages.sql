-- Voyages CRUD ------------------------------------------------------------

-- name: CreateVoyage :one
INSERT INTO shipman.voyages (
    charter_detail_id,
    voyage_number,
    vessel_name,
    departure_port,
    arrival_port,
    planned_departure_at,
    planned_arrival_at,
    actual_departure_at,
    actual_arrival_at,
    distance_nm,
    time_at_sea_hours,
    fuel_consumed_mt,
    fuel_type,
    weather_summary,
    status,
    notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
    COALESCE($15, 'planned'), $16
)
RETURNING *;

-- name: GetVoyage :one
SELECT *
FROM shipman.voyages
WHERE id = $1;

-- name: ListVoyagesForCharter :many
SELECT *
FROM shipman.voyages
WHERE charter_detail_id = $1
ORDER BY planned_departure_at NULLS LAST;

-- name: UpdateVoyageProgress :one
UPDATE shipman.voyages
SET
    actual_departure_at = COALESCE($2, actual_departure_at),
    actual_arrival_at = COALESCE($3, actual_arrival_at),
    distance_nm = COALESCE($4, distance_nm),
    time_at_sea_hours = COALESCE($5, time_at_sea_hours),
    fuel_consumed_mt = COALESCE($6, fuel_consumed_mt),
    fuel_type = COALESCE($7, fuel_type),
    weather_summary = COALESCE($8, weather_summary),
    status = COALESCE($9, status),
    notes = COALESCE($10, notes),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteVoyage :exec
DELETE FROM shipman.voyages
WHERE id = $1;

