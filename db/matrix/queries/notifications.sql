-- name: GetUserNotifications :many
SELECT DISTINCT ON(n.id) n.from_matrix_user_id,
    ms.display_name,
    ms.avatar_url,
    n.relates_to_event_id,
    n.event_id,
    n.thread_event_id,
    n.type,
    n.body,
    n.room_alias,
    n.created_at,
    n.read
FROM
notifications n
LEFT JOIN membership_state ms ON ms.user_id = n.from_matrix_user_id
WHERE n.for_matrix_user_id = $1 
ORDER BY n.id, n.created_at DESC
LIMIT 30;

-- name: MarkAsRead :exec
UPDATE notifications SET read_at = now(), read = true
WHERE for_matrix_user_id = $1;

-- name: CreateNotification :one
INSERT INTO notifications (
    for_matrix_user_id, 
    from_matrix_user_id, 
    room_id, 
    relates_to_event_id,
    thread_event_id,
    event_id,
    type, 
    body,
    room_alias
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

