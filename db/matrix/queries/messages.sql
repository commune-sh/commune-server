-- name: GetSpaceMessages :many
SELECT ej.event_id, 
    ej.json, 
    ud.display_name,
    ud.avatar_url,
    aliases.room_alias,
    RIGHT(events.event_id, 11) as slug,
    COALESCE(rc.count, 0) as replies,
    COALESCE(array_agg(json_build_object('key', re.aggregation_key, 'senders', re.senders)) FILTER (WHERE re.aggregation_key is not null), null) as reactions,
    ed.json::jsonb->'content'->>'m.new_content' as edited,
    COALESCE(NULLIF(ed.json::jsonb->>'origin_server_ts', '')::BIGINT, 0) as edited_on
FROM event_json ej
LEFT JOIN events on events.event_id = ej.event_id
LEFT JOIN aliases ON aliases.room_id = ej.room_id
LEFT JOIN membership_state ud ON ud.user_id = events.sender
    AND ud.room_id = ej.room_id
LEFT JOIN event_reactions re ON re.relates_to_id = ej.event_id
LEFT JOIN reply_count rc ON rc.relates_to_id = ej.event_id
LEFT JOIN redactions ON redactions.redacts = ej.event_id
LEFT JOIN (
	SELECT DISTINCT ON(evr.relates_to_id) ejs.json, evr.relates_to_id
	FROM event_json ejs
	JOIN event_relations evr ON evr.event_id = ejs.event_id
	JOIN events evs ON evr.event_id = evs.event_id
	AND evr.relation_type = 'm.replace'
	GROUP BY evr.relates_to_id, ejs.event_id, ejs.json, evs.origin_server_ts
	ORDER BY evr.relates_to_id, evs.origin_server_ts DESC
) ed ON ed.relates_to_id = ej.event_id
WHERE ej.room_id = $1
AND NOT EXISTS (SELECT FROM event_relations WHERE event_id = ej.event_id)
AND (events.origin_server_ts < sqlc.narg('origin_server_ts') OR sqlc.narg('origin_server_ts') IS NULL)
AND (events.origin_server_ts > sqlc.narg('after') OR sqlc.narg('after') IS NULL)
AND (ej.json::jsonb->'content'->>'topic' = sqlc.narg('topic') OR sqlc.narg('topic') IS NULL)
AND redactions.redacts is null
GROUP BY
    ej.event_id, 
    ed.json,
    events.event_id, 
    rc.count,
    ej.json,
    ud.display_name,
    ud.avatar_url,
    aliases.room_alias,
    events.origin_server_ts
ORDER BY CASE
    WHEN @order_by::text = 'ASC' THEN events.origin_server_ts 
END ASC, CASE 
    WHEN @order_by::text = 'DESC' THEN events.origin_server_ts 
END DESC, CASE
    WHEN @order_by::text = '' THEN events.origin_server_ts 
END DESC
LIMIT 50;



