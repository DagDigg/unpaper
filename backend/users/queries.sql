-- name: CreateUser :one
INSERT INTO users (id, given_name, family_name, email, password, username)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetUser :one
SELECT * 
FROM users
WHERE id = $1;

-- name: GetUserByUsername :one
SELECT * 
FROM users
WHERE username = $1;

-- name: DeleteUser :exec
DELETE
FROM users
WHERE id = $1;

-- name: GetPassword :one
SELECT password 
FROM users
WHERE id = $1;

-- name: GetEmail :one
SELECT email
FROM users
WHERE id = $1;

-- name: GetUserIDFromEmail :one
SELECT id
FROM users
WHERE email = $1;

-- name: UpdatePassword :exec
UPDATE users
SET password = $2
WHERE id = $1;

-- name: UpdatePasswordChangedAt :exec
UPDATE users
SET password_changed_at = $1
WHERE id = $2;

-- name: VerifyEmail :exec
UPDATE users
SET email_verified = TRUE
WHERE id = $1;

-- name: GetEmailVerified :one
SELECT email_verified
FROM users
WHERE id = $1;

-- name: UpdateUsername :one
UPDATE users
SET username = $1
WHERE id = $2
RETURNING *;

-- name: GetUserSuggestions :many
SELECT *
FROM users
WHERE (LOWER(username)) LIKE (LOWER($1))
LIMIT 4;

-- name: UserIDExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE id = $1);

-- name: EmailExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE (LOWER(email)) = $1);

-- name: UsernameExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE (LOWER(username)) = $1);