-- name: GetSpaceMessages :many
SELECT ej.event_id, 
    ej.json, 
    ud.display_name,
    ud.avatar_url,
    aliases.room_alias,
    RIGHT(events.event_id, 11) as slug,
    COALESCE(array_agg(json_build_object('key', re.aggregation_key, 'url', CASE WHEN re.url IS NOT NULL THEN re.url ELSE NULL END, 'senders', re.senders)) FILTER (WHERE re.aggregation_key is not null), null) as reactions,
    ed.json::jsonb->'content'->>'m.new_content' as edited,
    COALESCE(NULLIF(ed.json::jsonb->>'origin_server_ts', '')::BIGINT, 0) as edited_on,
    cast(prev.content as jsonb) as prev_content,
    CASE WHEN redactions.redacts IS NOT NULL THEN true ELSE false END as redacted,
    evt.replies,
    evt.last_reply as last_thread_reply
FROM event_json ej
LEFT JOIN events on events.event_id = ej.event_id
LEFT JOIN aliases ON aliases.room_id = ej.room_id
LEFT JOIN membership_state ud ON ud.user_id = events.sender
    AND ud.room_id = ej.room_id
LEFT JOIN event_reactions re ON re.relates_to_id = ej.event_id
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
LEFT JOIN (
    SELECT event_json.event_id,
        event_json.json::jsonb->>'content' as content
    FROM event_json
) prev ON prev.event_id = ej.json::jsonb->'unsigned'->>'replaces_state'
LEFT JOIN event_threads evt ON evt.event_id = ej.event_id
WHERE ej.room_id = $1
AND NOT EXISTS (SELECT FROM event_relations WHERE event_id = ej.event_id)
AND (events.origin_server_ts < sqlc.narg('origin_server_ts') OR sqlc.narg('origin_server_ts') IS NULL)
AND (events.origin_server_ts > sqlc.narg('after') OR sqlc.narg('after') IS NULL)
AND (ej.json::jsonb->'content'->>'topic' = sqlc.narg('topic') OR sqlc.narg('topic') IS NULL)
GROUP BY
    ej.event_id, 
    ed.json,
    events.event_id, 
    ej.json,
    ud.display_name,
    ud.avatar_url,
    aliases.room_alias,
    events.origin_server_ts,
    prev.content,
    redactions.redacts,
    evt.replies,
    evt.last_reply
ORDER BY CASE
    WHEN @order_by::text = 'ASC' THEN events.origin_server_ts 
END ASC, CASE 
    WHEN @order_by::text = 'DESC' THEN events.origin_server_ts 
END DESC, CASE
    WHEN @order_by::text = '' THEN events.origin_server_ts 
END DESC
LIMIT 50;



-- name: GetEventThread :many
SELECT ej.event_id, 
    ej.json, 
    ud.display_name,
    ud.avatar_url,
    aliases.room_alias,
    RIGHT(events.event_id, 11) as slug,
    COALESCE(array_agg(json_build_object('key', re.aggregation_key, 'url', CASE WHEN re.url IS NOT NULL THEN re.url ELSE NULL END, 'senders', re.senders)) FILTER (WHERE re.aggregation_key is not null), null) as reactions,
    ed.json::jsonb->'content'->>'m.new_content' as edited,
    COALESCE(NULLIF(ed.json::jsonb->>'origin_server_ts', '')::BIGINT, 0) as edited_on,
    cast(prev.content as jsonb) as prev_content,
    CASE WHEN redactions.redacts IS NOT NULL THEN true ELSE false END as redacted
FROM event_json ej
JOIN event_relations evre ON evre.event_id = ej.event_id
    AND evre.relation_type = 'm.thread'
LEFT JOIN events on events.event_id = ej.event_id
LEFT JOIN aliases ON aliases.room_id = ej.room_id
LEFT JOIN membership_state ud ON ud.user_id = events.sender
    AND ud.room_id = ej.room_id
LEFT JOIN event_reactions re ON re.relates_to_id = ej.event_id
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
LEFT JOIN (
    SELECT event_json.event_id,
        event_json.json::jsonb->>'content' as content
    FROM event_json
) prev ON prev.event_id = ej.json::jsonb->'unsigned'->>'replaces_state'
WHERE evre.relates_to_id = $1
AND events.type = 'm.room.message'
GROUP BY
    ej.event_id, 
    ed.json,
    events.event_id, 
    ej.json,
    ud.display_name,
    ud.avatar_url,
    aliases.room_alias,
    events.origin_server_ts,
    prev.content,
    redactions.redacts
ORDER BY events.origin_server_ts ASC
LIMIT 100;



