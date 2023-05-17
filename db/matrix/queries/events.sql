-- name: GetEvent :one
SELECT event_json.event_id, event_json.json FROM event_json
LEFT JOIN events on events.event_id = event_json.event_id
LEFT JOIN room_aliases ON room_aliases.room_id = event_json.room_id
WHERE events.sender = $1 
AND events.slug = $2
AND room_aliases.room_alias = $3 LIMIT 1;

-- name: GetUserEvents :many
SELECT event_json.event_id, event_json.json, events.slug FROM event_json
LEFT JOIN events on events.event_id = event_json.event_id
LEFT JOIN room_aliases ON room_aliases.room_id = event_json.room_id
WHERE events.sender = $1 
AND room_aliases.room_alias = $2
AND events.type = 'm.room.message'
ORDER BY events.origin_server_ts DESC LIMIT 100;



-- name: GetSpaceEvents :many
SELECT ej.event_id, 
    ej.json, 
    ud.display_name,
    ud.avatar_url,
    RIGHT(events.event_id, 11) as slug,
    COALESCE(rc.count, 0) as replies,
    COALESCE(array_agg(json_build_object('key', re.aggregation_key, 'senders', re.senders)) FILTER (WHERE re.aggregation_key is not null), null) as reactions
FROM event_json ej
LEFT JOIN events on events.event_id = ej.event_id
LEFT JOIN room_aliases ON room_aliases.room_id = ej.room_id
LEFT JOIN user_directory ud ON ud.user_id = events.sender
LEFT JOIN event_reactions re ON re.relates_to_id = ej.event_id
LEFT JOIN reply_count rc ON rc.relates_to_id = ej.event_id
LEFT JOIN redactions ON redactions.redacts = ej.event_id
WHERE ej.room_id = $1
AND events.type = 'm.room.message'
AND NOT EXISTS (SELECT FROM event_relations WHERE event_id = ej.event_id)
AND events.origin_server_ts < $2
AND redactions.redacts is null
GROUP BY
    ej.event_id, 
    events.event_id, 
    rc.count,
    ej.json,
    ud.display_name,
    ud.avatar_url,
    events.origin_server_ts
ORDER BY events.origin_server_ts DESC LIMIT 30;





-- name: GetSpaceEvent :one
SELECT ej.event_id, 
    ej.json,
    ud.display_name,
    ud.avatar_url,
    COALESCE(rc.count, 0) as replies,
    COALESCE(array_agg(json_build_object('key', re.aggregation_key, 'senders', re.senders)) FILTER (WHERE re.aggregation_key is not null), null) as reactions
FROM event_json ej
LEFT JOIN events on events.event_id = ej.event_id
LEFT JOIN user_directory ud ON ud.user_id = events.sender
LEFT JOIN room_aliases ON room_aliases.room_id = ej.room_id
LEFT JOIN event_reactions re ON re.relates_to_id = ej.event_id
LEFT JOIN reply_count rc ON rc.relates_to_id = ej.event_id
LEFT JOIN redactions ON redactions.redacts = ej.event_id
WHERE RIGHT(events.event_id, 11) = $1
AND redactions.redacts is null
GROUP BY
    ej.event_id, 
    events.event_id, 
    ej.json,
    ud.display_name,
    ud.avatar_url,
    rc.count
LIMIT 1;


-- name: GetSpaceEventReplies :many
SELECT ej.event_id, 
    ej.json, 
    ud.display_name,
    ud.avatar_url,
    RIGHT(events.event_id, 11) as slug,
    COALESCE(array_agg(json_build_object('key', re.aggregation_key, 'senders', re.senders)) FILTER (WHERE re.aggregation_key is not null), null) as reactions
FROM event_json ej
LEFT JOIN events on events.event_id = ej.event_id
LEFT JOIN user_directory ud ON ud.user_id = events.sender
LEFT JOIN event_relations ON event_relations.event_id = ej.event_id
LEFT JOIN event_reactions re ON re.relates_to_id = ej.event_id
LEFT JOIN redactions ON redactions.redacts = ej.event_id
WHERE events.type = 'm.room.message'
AND event_relations.relates_to_id = $1
AND redactions.redacts is null
GROUP BY
    ej.event_id, 
    ej.json,
    events.event_id,
    ud.display_name,
    ud.avatar_url,
    events.origin_server_ts
ORDER BY events.origin_server_ts DESC LIMIT 1000;




-- name: GetEvents :many
SELECT ej.event_id, 
    ej.json, 
    room_aliases.room_alias,
    ud.display_name,
    ud.avatar_url,
    RIGHT(events.event_id, 11) as slug,
    COALESCE(rc.count, 0) as replies,
    COALESCE(array_agg(json_build_object('key', re.aggregation_key, 'senders', re.senders)) FILTER (WHERE re.aggregation_key is not null), null) as reactions
FROM event_json ej
LEFT JOIN events on events.event_id = ej.event_id
LEFT JOIN user_directory ud ON ud.user_id = events.sender
LEFT JOIN room_aliases ON room_aliases.room_id = ej.room_id
LEFT JOIN event_reactions re ON re.relates_to_id = ej.event_id
LEFT JOIN reply_count rc ON rc.relates_to_id = ej.event_id
LEFT JOIN redactions ON redactions.redacts = ej.event_id
WHERE events.type = 'm.room.message'
AND NOT EXISTS (SELECT FROM event_relations WHERE event_id = ej.event_id)
AND room_aliases.room_alias is not null
AND events.origin_server_ts < $1
AND redactions.redacts is null
GROUP BY
    ej.event_id, 
    events.event_id, 
    rc.count,
    ej.json,
    ud.display_name,
    ud.avatar_url,
    events.origin_server_ts,
    room_aliases.room_alias
ORDER BY events.origin_server_ts DESC LIMIT 30;


-- name: GetUserFeedEvents :many
SELECT ej.event_id, 
    ej.json, 
    room_aliases.room_alias,
    ud.display_name,
    ud.avatar_url,
    RIGHT(events.event_id, 11) as slug,
    COALESCE(rc.count, 0) as replies,
    COALESCE(array_agg(json_build_object('key', re.aggregation_key, 'senders', re.senders)) FILTER (WHERE re.aggregation_key is not null), null) as reactions
FROM event_json ej
LEFT JOIN events on events.event_id = ej.event_id
LEFT JOIN user_directory ud ON ud.user_id = events.sender
LEFT JOIN room_aliases ON room_aliases.room_id = ej.room_id
LEFT JOIN event_reactions re ON re.relates_to_id = ej.event_id
LEFT JOIN reply_count rc ON rc.relates_to_id = ej.event_id
LEFT JOIN redactions ON redactions.redacts = ej.event_id
JOIN membership_state ms 
    ON ms.room_id = ej.room_id 
    AND ms.user_id = $2
    AND ms.membership = 'join'
WHERE events.type = 'm.room.message'
AND NOT EXISTS (SELECT FROM event_relations WHERE event_id = ej.event_id)
AND room_aliases.room_alias is not null
AND events.origin_server_ts < $1
AND redactions.redacts is null
GROUP BY
    ej.event_id, 
    events.event_id, 
    rc.count,
    ej.json,
    ud.display_name,
    ud.avatar_url,
    events.origin_server_ts,
    room_aliases.room_alias
ORDER BY events.origin_server_ts DESC LIMIT 30;

