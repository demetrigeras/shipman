-- Users CRUD ---------------------------------------------------------------

-- name: CreateUser :one
INSERT INTO shipman.users (email, password_hash, full_name, role)
VALUES ($1, $2, $3, COALESCE($4, 'user'))
RETURNING *;

-- name: GetUser :one
SELECT *
FROM shipman.users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT *
FROM shipman.users
WHERE email = $1;

-- name: ListUsers :many
SELECT *
FROM shipman.users
ORDER BY created_at DESC;

-- name: UpdateUser :one
UPDATE shipman.users
SET
    email = COALESCE($2, email),
    password_hash = COALESCE($3, password_hash),
    full_name = COALESCE($4, full_name),
    role = COALESCE($5, role),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM shipman.users
WHERE id = $1;

