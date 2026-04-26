-- name: ListCategories :many
SELECT * FROM categories ORDER BY type, name;

-- name: GetCategory :one
SELECT * FROM categories WHERE id = $1;

-- name: CreateCategory :one
INSERT INTO categories (name, type, parent_id)
VALUES ($1, $2, $3)
RETURNING *;
