-- name: DoesSpaceExist :one
SELECT exists(select 1 from room_aliases where room_alias = $1 OR slug = $2);

-- name: GetUserSpaceID :one
SELECT ra.room_id
FROM room_aliases ra
WHERE ra.room_alias = $1
AND ra.creator = $2;

-- name: GetAllCommunities :many
SELECT room_aliases.room_alias,
    room_aliases.room_id,
    room_aliases.creator,
    rooms.is_public,
    rooms.room_version
FROM room_aliases
LEFT JOIN rooms ON room_aliases.room_id = rooms.room_id;


-- name: GetSpaceState :one
SELECT ra.room_id, rm.members, ev.origin_server_ts, ev.sender as owner,
	jsonb_build_object('name', rs.name,'topic', rs.topic, 'avatar', rs.avatar, 'header', rs.header) as state,
	COALESCE(array_agg(json_build_object('room_id', ch.room_id, 'name', ch.name, 'topic', ch.topic, 'avatar', ch.avatar, 'header', ch.header, 'slug', ch.slug)) FILTER (WHERE ch.room_id IS NOT NULL), null) as children
FROM room_aliases ra 
LEFT JOIN (
	SELECT * FROM room_state
) as rs ON rs.room_id = ra.room_id
LEFT JOIN (
	SELECT * FROM room_state JOIN space_children ON space_children.child_room_id = room_state.room_id
) as ch ON ch.parent_room_id = ra.room_id
LEFT JOIN events ev ON ev.room_id = ra.room_id and ev.type = 'm.room.create'
LEFT JOIN room_members rm ON rm.room_id = ra.room_id
WHERE ra.room_alias = $1
GROUP BY ra.room_id, rm.members, ev.origin_server_ts, ev.sender, rs.name, rs.topic, rs.avatar, rs.header;


-- name: GetSpaceChildren :many
WITH sel AS (
SELECT ej.room_id as child_room_id, ej.json::jsonb->>'state_key' as parent_room_id
FROM event_json as ej
LEFT JOIN room_aliases ON room_aliases.room_id = ej.json::jsonb->>'state_key'
WHERE ej.json::jsonb->>'type' = 'm.space.parent'
AND room_aliases.room_alias = $1
) select DISTINCT ON (sel.child_room_id) sel.child_room_id, event_json.json::jsonb->'content'->>'name' as name, events.origin_server_ts
FROM event_json JOIN sel on sel.child_room_id = event_json.room_id
LEFT JOIN events on events.event_id = event_json.event_id
WHERE event_json.json::jsonb->>'type' = 'm.room.name' 
ORDER BY sel.child_room_id, events.origin_server_ts DESC;
