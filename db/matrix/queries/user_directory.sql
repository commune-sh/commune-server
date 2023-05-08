-- name: CreateUserDirectory :one
INSERT INTO user_directory (
    user_id, display_name
) VALUES (
  $1, $2
)
RETURNING user_id;
