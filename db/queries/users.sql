-- name: GetCredentials :one
SELECT id, username, email, verified, created_at FROM users
WHERE (username = $1 OR email = $1) AND deleted = false LIMIT 1;

-- name: GetUser :one
SELECT * FROM users
WHERE username = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: IsVerifed :one
SELECT verified FROM users
WHERE matrix_user_id = $1 OR username = $1 LIMIT 1;

-- name: DoesEmailExist :one
SELECT exists(select 1 from users where email = $1 AND deleted = false);

-- name: DoesUsernameExist :one
SELECT exists(select 1 from users where username = $1 AND deleted = false);

-- name: HasUserVerifiedEmail :one
SELECT exists(select 1 from users where username = $1 AND verified = true);

-- name: ListUsers :many
SELECT * FROM users
ORDER BY username;

-- name: CreateUser :one
INSERT INTO users (
  matrix_user_id, username
) VALUES (
  $1, $2
)
RETURNING id, created_at;

-- name: VerifyEmail :exec
UPDATE users SET email = $1, verified = true WHERE matrix_user_id = $2;

-- name: DeactivateUser :exec
UPDATE users SET deactivated_at = NOW() AND deactivated = true WHERE id = $1;

-- name: ReactivateUser :exec
UPDATE users SET reactivated_at = NOW() AND deactivated = false WHERE id = $1;

-- name: DeleteUser :exec
UPDATE users SET deleted_at = NOW() AND deleted = true WHERE id = $1;
