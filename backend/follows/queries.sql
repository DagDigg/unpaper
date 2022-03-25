-- name: FollowUser :one
WITH f AS (
	INSERT INTO follows (follower_user_id, following_user_id, follow_date)
	VALUES ($1, $2, $3)
	ON CONFLICT (follower_user_id, following_user_id)
	DO UPDATE SET follow_date = EXCLUDED.follow_date
	RETURNING *
)
SELECT f.*, u.*
FROM f
JOIN users u ON f.following_user_id = u.id;

-- name: UnfollowUser :one
UPDATE follows
SET unfollow_date=$1
FROM users u
WHERE 
following_user_id = u.id
AND follower_user_id=$2
AND following_user_id=$3
RETURNING *;

-- name: GetFollowing :many
SELECT f.*, u.* FROM follows f
JOIN users u ON f.following_user_id = u.id
WHERE follower_user_id=$1
AND follow_date > unfollow_date OR unfollow_date IS NULL;

-- name: GetFollowers :many
SELECT f.*, u.* FROM follows f
JOIN users u ON f.follower_user_id = u.id
WHERE following_user_id=$1
AND follow_date > unfollow_date OR unfollow_date IS NULL;

-- name: GetFollowersCount :one
SELECT COUNT(*) FROM follows 
WHERE following_user_id=$1
AND follow_date > unfollow_date  OR unfollow_date IS NULL;

-- name: GetFollowingCount :one
SELECT COUNT(*) FROM follows 
WHERE follower_user_id=$1
AND follow_date > unfollow_date  OR unfollow_date IS NULL;

-- name: IsFollowingUser :one
SELECT EXISTS(SELECT * FROM follows WHERE follower_user_id=$1 AND following_user_id=$2 AND (follow_date > unfollow_date OR unfollow_date IS NULL));