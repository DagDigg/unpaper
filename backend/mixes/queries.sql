-- name: GetUserMixes :many
SELECT * FROM mixes WHERE user_id=$1;

-- name: CreateUserMix :one
INSERT INTO mixes (id, user_id, post_ids, background, requested_at, category, title)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: DeleteUserMixes :exec
DELETE FROM mixes WHERE user_id = $1;