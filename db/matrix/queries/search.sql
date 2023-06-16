-- name: SearchEvents :many
SELECT ej.event_id, 
    ej.json, 
    ud.display_name,
    ud.avatar_url,
    aliases.room_alias,
    RIGHT(events.event_id, 11) as slug
FROM event_json ej
JOIN search ON search.event_id = ej.event_id
LEFT JOIN events on events.event_id = ej.event_id
LEFT JOIN aliases ON aliases.room_id = ej.room_id
LEFT JOIN user_directory ud ON ud.user_id = events.sender
LEFT JOIN redactions ON redactions.redacts = ej.event_id
WHERE search.title_vec @@ $1::tsquery
OR search.body_vec @@ $1::tsquery
AND events.type = 'm.room.message'
AND NOT EXISTS (SELECT FROM event_relations WHERE event_id = ej.event_id)
AND redactions.redacts is null
GROUP BY
    ej.event_id, 
    events.event_id, 
    ej.json,
    ud.display_name,
    ud.avatar_url,
    aliases.room_alias,
    events.origin_server_ts
ORDER BY events.origin_server_ts DESC
LIMIT 10;




