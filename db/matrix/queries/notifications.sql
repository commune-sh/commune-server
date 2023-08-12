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
	CASE WHEN er.relation_type = 'm.annotation' THEN 
        CASE WHEN ej.json::jsonb->'content'->'m.relates_to'->>'url' IS NOT NULL
            THEN ej.json::jsonb->'content'->'m.relates_to'->>'url'
        ELSE ej.json::jsonb->'content'->'m.relates_to'->>'key' END
    ELSE ej.json::jsonb->'content'->>'body' END as body, 
	ej.json::jsonb->'content'->'m.relates_to'->>'thread_event_id' as thread_event_id,
	ev.origin_server_ts as created_at, 
    events.type as event_type,
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
WITH NOTS AS (
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
	CASE WHEN er.relation_type = 'm.annotation' THEN 
        CASE WHEN ej.json::jsonb->'content'->'m.relates_to'->>'url' IS NOT NULL
            THEN ej.json::jsonb->'content'->'m.relates_to'->>'url'
        ELSE ej.json::jsonb->'content'->'m.relates_to'->>'key' END
    ELSE ej.json::jsonb->'content'->>'body' END as body, 
	ej.json::jsonb->'content'->'m.relates_to'->>'thread_event_id' as thread_event_id,
	ev.origin_server_ts as created_at, 
    events.type as event_type,
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
),
FOL AS (
SELECT 'space.follow' as type,
	ms.user_id as from_matrix_user_id,
	'' as body,
	'' as thread_event_id,
	ms.origin_server_ts as created_at,
    '' as event_type,
	'' as event_id,
	'' as relates_to_event_id,
	ms.display_name,
	ms.avatar_url,
	'' as room_alias,
    CASE WHEN ms.origin_server_ts > $2 THEN false ELSE true END as read
FROM membership_state ms 
WHERE ms.user_id != $1
AND ms.room_id = $3
AND ms.membership = 'join'
)
SELECT * FROM NOTS 
UNION ALL 
SELECT * FROM FOL
ORDER BY created_at DESC
LIMIT 50;
