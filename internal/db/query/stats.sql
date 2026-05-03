-- name: GetMonthStats :one
SELECT
  COUNT(*)::int                          AS count,
  COALESCE(SUM(base_amount), 0)::numeric AS total
FROM expenses
WHERE base_currency = $1
  AND date >= $2
  AND date < $3
  AND (sqlc.narg(user_id)::uuid IS NULL OR user_id = sqlc.narg(user_id)::uuid)
  AND is_refund = false;

-- name: GetMonthStatsByYear :many
SELECT
  EXTRACT(MONTH FROM date)::int          AS month,
  COUNT(*)::int                          AS count,
  COALESCE(SUM(base_amount), 0)::numeric AS total
FROM expenses
WHERE base_currency = $1
  AND date >= $2
  AND date < $3
  AND (sqlc.narg(user_id)::uuid IS NULL OR user_id = sqlc.narg(user_id)::uuid)
  AND is_refund = false
  AND base_amount IS NOT NULL
GROUP BY EXTRACT(MONTH FROM date)
ORDER BY month;

-- name: GetAllTimeMonthlyStats :many
SELECT
  EXTRACT(YEAR  FROM date)::int          AS year,
  EXTRACT(MONTH FROM date)::int          AS month,
  COUNT(*)::int                          AS count,
  COALESCE(SUM(base_amount), 0)::numeric AS total
FROM expenses
WHERE base_currency = $1
  AND (sqlc.narg(user_id)::uuid IS NULL OR user_id = sqlc.narg(user_id)::uuid)
  AND is_refund = false
  AND base_amount IS NOT NULL
GROUP BY year, month
ORDER BY year, month;