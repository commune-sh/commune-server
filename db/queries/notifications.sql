-- name: CreateNotification :one
INSERT INTO notifications (
    matrix_user_id, type, content
) VALUES (
  $1, $2, $3
)
RETURNING created_at;

