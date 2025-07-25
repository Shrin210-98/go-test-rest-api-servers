-- name: GetAuthor :one
SELECT * FROM authors WHERE id = $1 LIMIT 1;

-- name: ListAuthors :many
SELECT * FROM authors ORDER BY name;

-- name: CreateAuthor :one
INSERT INTO authors (name, bio, book_name) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateAuthor :exec
UPDATE authors set name = $2, bio = $3 WHERE id = $1;

-- name: DeleteAuthor :exec
DELETE FROM authors WHERE id = $1;

-- name: CountAuthors :one
SELECT COUNT(*) FROM authors;