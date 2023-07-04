-- name: GetEvent :one
SELECT event_json.event_id, event_json.json FROM event_json
LEFT JOIN events on events.event_id = event_json.event_id
LEFT JOIN aliases ON aliases.room_id = event_json.room_id
WHERE events.sender = $1 
AND events.slug = $2
AND aliases.room_alias = $3 LIMIT 1;

-- name: GetUserEvents :many
SELECT event_json.event_id, event_json.json, events.slug FROM event_json
LEFT JOIN events on events.event_id = event_json.event_id
LEFT JOIN aliases ON aliases.room_id = event_json.room_id
WHERE events.sender = $1 
AND aliases.room_alias = $2
AND events.type = 'space.board.post'
ORDER BY events.origin_server_ts DESC LIMIT 100;


-- name: GetPinnedEvents :one
SELECT ej.json::json->'content'->>'pinned'::text AS events
FROM current_state_events cse
JOIN event_json ej ON ej.event_id = cse.event_id
WHERE cse.type = 'm.room.pinned_events' 
AND cse.room_id = $1;

-- name: GetSpaceEvents :many
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
AND events.type = 'space.board.post'
AND NOT EXISTS (SELECT FROM event_relations WHERE event_id = ej.event_id 
AND relation_type != 'm.reference')
AND (events.origin_server_ts < sqlc.narg('origin_server_ts') OR sqlc.narg('origin_server_ts') IS NULL)
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
LIMIT 30;





-- name: GetSpaceEvent :one
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
LEFT JOIN membership_state ud ON ud.user_id = events.sender
    AND ud.room_id = ej.room_id
LEFT JOIN aliases ON aliases.room_id = ej.room_id
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
WHERE RIGHT(events.event_id, 11) = $1
AND redactions.redacts is null
GROUP BY
    ej.event_id, 
    ed.json,
    events.event_id, 
    ej.json,
    ud.display_name,
    ud.avatar_url,
    aliases.room_alias,
    rc.count
LIMIT 1;


-- name: GetSpaceEventReplies :many
WITH RECURSIVE recursive_events AS (
    SELECT event_id, relates_to_id
    FROM event_relations
    WHERE RIGHT(event_relations.relates_to_id, 11) = sqlc.narg('slug')
    AND event_relations.relation_type = 'm.nested_reply'

    UNION

    SELECT er.event_id, er.relates_to_id
    FROM event_relations er
    INNER JOIN recursive_events re ON re.event_id = er.relates_to_id
    WHERE er.relation_type != 'm.replace'
)
SELECT DISTINCT ON (ej.event_id) ej.event_id, 
    ej.json, 
    rev.relates_to_id as in_reply_to,
    ud.display_name,
    ud.avatar_url,
    aliases.room_alias,
    RIGHT(events.event_id, 11) as slug,
    COALESCE(array_agg(json_build_object('key', re.aggregation_key, 'senders', re.senders)) FILTER (WHERE re.aggregation_key is not null), null) as reactions,
    ed.json::jsonb->'content'->>'m.new_content' as edited,
    COALESCE(NULLIF(ed.json::jsonb->>'origin_server_ts', '')::BIGINT, 0) as edited_on,
    votes.upvotes, 
    votes.downvotes,
    CASE WHEN upvoted.sender IS NOT NULL THEN TRUE ELSE FALSE END as upvoted,
    CASE WHEN downvoted.sender IS NOT NULL THEN TRUE ELSE FALSE END as downvoted
FROM event_json ej
JOIN recursive_events rev ON rev.event_id = ej.event_id
LEFT JOIN events on events.event_id = ej.event_id
LEFT JOIN membership_state ud ON ud.user_id = events.sender
    AND ud.room_id = ej.room_id
LEFT JOIN aliases ON aliases.room_id = ej.room_id
LEFT JOIN event_reactions re ON re.relates_to_id = ej.event_id
LEFT JOIN event_votes votes ON votes.relates_to_id = ej.event_id
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
	SELECT er.relates_to_id, evts.sender, er.aggregation_key
	FROM event_relations er
	JOIN events evts ON evts.event_id = er.event_id
	WHERE er.relation_type = 'm.annotation' AND er.aggregation_key = 'upvote' 
	AND evts.sender = sqlc.narg('sender')::text
) upvoted ON upvoted.relates_to_id = ej.event_id
LEFT JOIN (
	SELECT er.relates_to_id, evts.sender, er.aggregation_key
	FROM event_relations er
	JOIN events evts ON evts.event_id = er.event_id
	WHERE er.relation_type = 'm.annotation' AND er.aggregation_key = 'downvote' 
	AND evts.sender = sqlc.narg('sender')::text
) downvoted ON downvoted.relates_to_id = ej.event_id
WHERE events.type = 'space.board.post.reply'
AND redactions.redacts is null
GROUP BY
    ej.event_id, 
    ej.json,
    ed.json,
    rev.relates_to_id,
    events.event_id,
    ud.display_name,
    ud.avatar_url,
    aliases.room_alias,
    votes.upvotes, 
    votes.downvotes,
    upvoted.sender, 
    downvoted.sender,
    events.origin_server_ts;




-- name: GetEvents :many
SELECT ej.event_id, 
    ej.json, 
    aliases.room_alias,
    ud.display_name,
    ud.avatar_url,
    RIGHT(events.event_id, 11) as slug,
    COALESCE(rc.count, 0) as replies,
    COALESCE(array_agg(json_build_object('key', re.aggregation_key, 'senders', re.senders)) FILTER (WHERE re.aggregation_key is not null), null) as reactions,
    ed.json::jsonb->'content'->>'m.new_content' as edited,
    COALESCE(NULLIF(ed.json::jsonb->>'origin_server_ts', '')::BIGINT, 0) as edited_on
FROM event_json ej
JOIN room_state rs ON rs.room_id = ej.room_id
LEFT JOIN events on events.event_id = ej.event_id
LEFT JOIN membership_state ud ON ud.user_id = events.sender
    AND ud.room_id = ej.room_id
LEFT JOIN aliases ON aliases.room_id = ej.room_id
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
WHERE events.type = 'space.board.post'
AND rs.do_not_index IS FALSE
AND NOT EXISTS (SELECT FROM event_relations WHERE event_id = ej.event_id)
AND aliases.room_alias is not null
AND events.origin_server_ts < $1
AND redactions.redacts is null
GROUP BY
    ej.event_id, 
    ed.json,
    events.event_id, 
    rc.count,
    ej.json,
    ud.display_name,
    ud.avatar_url,
    events.origin_server_ts,
    aliases.room_alias
ORDER BY events.origin_server_ts DESC LIMIT 30;


-- name: GetUserFeedEvents :many
SELECT ej.event_id, 
    ej.json, 
    aliases.room_alias,
    ud.display_name,
    ud.avatar_url,
    RIGHT(events.event_id, 11) as slug,
    COALESCE(rc.count, 0) as replies,
    COALESCE(array_agg(json_build_object('key', re.aggregation_key, 'senders', re.senders)) FILTER (WHERE re.aggregation_key is not null), null) as reactions,
    ed.json::jsonb->'content'->>'m.new_content' as edited,
    COALESCE(NULLIF(ed.json::jsonb->>'origin_server_ts', '')::BIGINT, 0) as edited_on
FROM event_json ej
LEFT JOIN events on events.event_id = ej.event_id
LEFT JOIN membership_state ud ON ud.user_id = events.sender
    AND ud.room_id = ej.room_id
LEFT JOIN aliases ON aliases.room_id = ej.room_id
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
JOIN membership_state ms 
    ON ms.room_id = ej.room_id 
    AND ms.user_id = $1
    AND ms.membership = 'join'
WHERE events.type = 'space.board.post'
AND NOT EXISTS (SELECT FROM event_relations WHERE event_id = ej.event_id)
AND aliases.room_alias is not null
AND (events.origin_server_ts < sqlc.narg('origin_server_ts') OR sqlc.narg('origin_server_ts') IS NULL)
AND redactions.redacts is null
GROUP BY
    ej.event_id, 
    ed.json,
    events.event_id, 
    rc.count,
    ej.json,
    ud.display_name,
    ud.avatar_url,
    events.origin_server_ts,
    aliases.room_alias
ORDER BY events.origin_server_ts DESC LIMIT 30;

