-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at;

-- name: CreateUser :one
INSERT INTO users (name) VALUES ($1) RETURNING *;
