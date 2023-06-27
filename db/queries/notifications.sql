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

