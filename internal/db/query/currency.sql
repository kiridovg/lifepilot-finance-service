-- name: ListCurrencies :many
SELECT * FROM currencies WHERE is_active = TRUE ORDER BY code;
