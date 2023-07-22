-- name: GetNotification :one
SELECT 
	CASE WHEN (er.relation_type = 'm.annotation' AND events.type = 'space.board.post.reply') THEN 'reply.reaction'
	ELSE CASE WHEN er.relation_type = 'm.annotation' THEN 'reaction' 
	ELSE
	    CASE WHEN events.type = 'space.board.post' THEN 'post.reply' 
	    ELSE
		CASE WHEN events.type = 'space.board.post.reply' THEN 'reply.reply'
		END
	    END
	END END as type,
	ev.sender as from_matrix_user_id, 
	events.sender as for_matrix_user_id, 
	CASE WHEN er.relation_type = 'm.annotation' THEN er.aggregation_key ELSE ej.json::jsonb->'content'->>'body' END as body, 
	ej.json::jsonb->'content'->'m.relates_to'->>'thread_event_id' as thread_event_id,
	ev.origin_server_ts as created_at, 
	ev.event_id, 
	er.relates_to_id as relates_to_event_id,
	ms.display_name,
	ms.avatar_url,
	aliases.room_alias,
    false as read
FROM event_json ej
JOIN event_relations er ON er.event_id = ej.event_id
JOIN events ON events.event_id = er.relates_to_id
JOIN events ev ON ev.event_id = er.event_id
JOIN aliases ON aliases.room_id = events.room_id
JOIN membership_state ms ON ms.user_id = ev.sender
WHERE ev.event_id = $1;




-- name: GetNotifications :many
SELECT DISTINCT ON (ev.origin_server_ts) 
	CASE WHEN (er.relation_type = 'm.annotation' AND events.type = 'space.board.post.reply') THEN 'reply.reaction'
	ELSE CASE WHEN er.relation_type = 'm.annotation' THEN 'reaction' 
	ELSE
	    CASE WHEN events.type = 'space.board.post' THEN 'post.reply' 
	    ELSE
		CASE WHEN events.type = 'space.board.post.reply' THEN 'reply.reply'
		END
	    END
	END END as type,
	ev.sender as from_matrix_user_id, 
	CASE WHEN er.relation_type = 'm.annotation' THEN er.aggregation_key ELSE ej.json::jsonb->'content'->>'body' END as body, 
	ej.json::jsonb->'content'->'m.relates_to'->>'thread_event_id' as thread_event_id,
	ev.origin_server_ts as created_at, 
	ev.event_id, 
	er.relates_to_id as relates_to_event_id,
	ms.display_name,
	ms.avatar_url,
	aliases.room_alias,
    CASE WHEN ev.origin_server_ts > $2 THEN false ELSE true END as read
FROM event_json ej
JOIN event_relations er ON er.event_id = ej.event_id
JOIN events ON events.event_id = er.relates_to_id
JOIN events ev ON ev.event_id = er.event_id
JOIN aliases ON aliases.room_id = events.room_id
JOIN membership_state ms ON ms.user_id = ev.sender
WHERE events.sender = $1
AND ev.sender != $1
AND ej.json::jsonb->'content'->'m.relates_to'->>'thread_event_id' is not NULL
AND (er.relation_type = 'm.annotation' OR er.relation_type = 'm.nested_reply')
GROUP BY ev.event_id, ev.sender, ej.json, ev.origin_server_ts, er.relates_to_id, ms.display_name, ms.avatar_url, aliases.room_alias, events.type, er.relation_type, er.aggregation_key
ORDER BY ev.origin_server_ts DESC LIMIT 30;

-- name: GetUserNotifications :many
SELECT DISTINCT ON(n.created_at) n.from_matrix_user_id,
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
ORDER BY n.created_at DESC
LIMIT 30;

-- name: MarkAsRead :exec
UPDATE notifications SET read_at = now(), read = true
WHERE for_matrix_user_id = $1;

-- name: CreateNotification :one
WITH INS AS (
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
    RETURNING *
)
SELECT INS.*,
    ms.display_name,
    ms.avatar_url
FROM INS
LEFT JOIN membership_state ms ON ms.user_id = INS.from_matrix_user_id;
