-- Ship Positions CRUD -----------------------------------------------------

-- name: CreateShipPosition :one
INSERT INTO shipman.ship_positions (
    voyage_id,
    recorded_at,
    latitude,
    longitude,
    speed_knots,
    heading,
    distance_logged_nm,
    fuel_remaining_mt,
    source,
    remarks
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, COALESCE($9, 'manual'), $10
)
RETURNING *;

-- name: ListShipPositions :many
SELECT *
FROM shipman.ship_positions
WHERE voyage_id = $1
ORDER BY recorded_at DESC
LIMIT $2;

-- name: DeleteShipPosition :exec
DELETE FROM shipman.ship_positions
WHERE id = $1;

