-- name: GetListsByOwnerID :many
SELECT * FROM lists
WHERE owner_user_id = $1;

-- name: GetListByID :one
SELECT * FROM lists
WHERE id = $1;

-- name: CreateList :one
INSERT INTO lists
(id, name, allowed_users, owner_user_id)
VALUES
($1, $2, $3, $4)
RETURNING *;

-- name: UpdateAllowedUsers :one
UPDATE lists
SET
allowed_users = $1
WHERE id = $2
RETURNING *;

-- name: UpdateName :one
UPDATE lists
SET name = $1
WHERE id = $2
RETURNING *;