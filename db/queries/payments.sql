-- Payments CRUD -----------------------------------------------------------

-- name: CreatePayment :one
INSERT INTO shipman.payments (
    charter_detail_id,
    voyage_id,
    category,
    due_date,
    paid_at,
    amount,
    currency,
    status,
    payment_method,
    reference,
    notes
) VALUES (
    $1, $2, COALESCE($3, 'general'), $4, $5, $6, COALESCE($7, 'USD'),
    COALESCE($8, 'pending'), $9, $10, $11
)
RETURNING *;

-- name: GetPayment :one
SELECT *
FROM shipman.payments
WHERE id = $1;

-- name: ListPaymentsForCharter :many
SELECT *
FROM shipman.payments
WHERE charter_detail_id = $1
ORDER BY due_date NULLS LAST, created_at DESC;

-- name: UpdatePaymentStatus :one
UPDATE shipman.payments
SET
    status = COALESCE($2, status),
    paid_at = COALESCE($3, paid_at),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeletePayment :exec
DELETE FROM shipman.payments
WHERE id = $1;

