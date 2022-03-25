-- name: CreateComment :one
INSERT INTO comments (id, message, audio, author, parent_id, likes, post_id, thread_type, thread_target_id)
VALUES ($1, $2, $3, $4, $5, 0, $6, $7, $8)
RETURNING *;

-- name: GetComments :many
SELECT * FROM comments c WHERE c.post_id = $1 AND c.thread_type = 'post'
UNION ALL
(SELECT * FROM comments z WHERE z.post_id = $1 AND thread_type != 'post' ORDER BY likes DESC);

-- name: LikeComment :one
UPDATE comments
SET
likes = likes + 1,
user_ids_who_likes = array_append(user_ids_who_likes,sqlc.arg(user_id)::VARCHAR(100))
WHERE id = sqlc.arg(id)::VARCHAR(100)
RETURNING *;

-- name: RemoveLikeFromComment :one
UPDATE comments
SET
likes = likes - 1,
user_ids_who_likes = array_remove(user_ids_who_likes,sqlc.arg(user_id)::VARCHAR(100))
WHERE id = sqlc.arg(id)::VARCHAR(100)
RETURNING *;


-- name: HasUserLikedComment :one
SELECT EXISTS(SELECT 1 FROM comments WHERE id=$1 AND $2::VARCHAR(100)=ANY(user_ids_who_likes::VARCHAR(100)[]));