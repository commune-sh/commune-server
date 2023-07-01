-- name: GetUserNotifications :many
SELECT * from 
notifications 
WHERE for_matrix_user_id = $1 
ORDER BY created_at DESC
LIMIT 30;

-- name: MarkAsRead :exec
UPDATE notifications SET read_at = now(), read = true
WHERE for_matrix_user_id = $1;

-- name: CreateNotification :one
INSERT INTO notifications (
    for_matrix_user_id, 
    from_matrix_user_id, 
    relates_to_event_id,
    thread_event_id,
    event_id,
    type, 
    body,
    room_alias
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING created_at;

