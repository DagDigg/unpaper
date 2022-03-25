-- name: CreatePost :one
INSERT INTO posts (id, author, message, audio, created_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPost :one
SELECT * from posts
WHERE id = $1;

-- name: GetPosts :many
SELECT * from posts; -- TODO: pagination

-- name: LikePost :one
UPDATE posts
SET
likes = likes + 1,
user_ids_who_likes = array_append(user_ids_who_likes,$1::VARCHAR(100))
WHERE id = $2
RETURNING *;

-- name: RemoveLikeFromPost :one
UPDATE posts
SET
likes = likes - 1,
user_ids_who_likes = array_remove(user_ids_who_likes,$1::VARCHAR(100))
WHERE id = $2
RETURNING *;

-- name: HasUserLikedPost :one
SELECT EXISTS(SELECT 1 FROM posts WHERE id=$1 AND $2::VARCHAR(100)=ANY(user_ids_who_likes::VARCHAR(100)[]));

-- name: GetTrendingTodayPosts :many
WITH p AS (
	SELECT * FROM posts
	WHERE created_at > current_timestamp - interval '1 day'
	ORDER BY likes DESC
	LIMIT 30
)
SELECT * FROM p 
ORDER BY RANDOM()
LIMIT 10;

-- name: GetTrendingTodayPostIDs :many
WITH p AS (
	SELECT id FROM posts
	WHERE created_at > current_timestamp - interval '1 day'
	ORDER BY likes DESC
	LIMIT 30
)
SELECT * FROM p 
ORDER BY RANDOM()
LIMIT 10;