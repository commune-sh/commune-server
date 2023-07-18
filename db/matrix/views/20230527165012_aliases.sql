DROP INDEX IF EXISTS aliases_idx;
DROP MATERIALIZED VIEW IF EXISTS aliases;
DROP TRIGGER aliases_mv_trigger on current_state_events;
DROP FUNCTION aliases_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS aliases AS 
WITH space_children AS(
    WITH sc AS (
	SELECT er.json::jsonb->'content'->>'alias' as parent_room_alias, rooms.room_id as parent_room_id, cse.state_key as child_room_id, trim(BOTH '-' FROM regexp_replace(unaccent(trim(sc.alias)), '[^a-z0-9\\_-]+', '-', 'gi')) as child_room_alias, events.origin_server_ts
        FROM rooms
	JOIN event_json er ON er.room_id = rooms.room_id AND er.json::jsonb->>'type' = 'm.room.canonical_alias'
        LEFT JOIN current_state_events as cse ON cse.room_id = rooms.room_id AND cse.type ='m.space.child'
        LEFT JOIN event_json ev ON ev.event_id = cse.event_id
        JOIN events ON events.event_id = ev.event_id
        LEFT JOIN (
        SELECT cs.room_id, COALESCE(ej.json::jsonb->'content'->>'name'::text, 'untitled') as alias FROM current_state_events cs 
        JOIN event_json ej ON ej.event_id = cs.event_id
        WHERE cs.type = 'm.room.name'
        ) as sc ON sc.room_id = cse.state_key
        WHERE ev.json::jsonb->'content'->>'via' is not null
        ORDER BY events.origin_server_ts DESC
    ) SELECT sc.parent_room_alias, sc.parent_room_id, sc.child_room_id,
    CASE WHEN ROW_NUMBER() OVER (PARTITION BY sc.parent_room_alias, sc.child_room_alias ORDER BY sc.parent_room_alias, sc.origin_server_ts) = 1 THEN sc.child_room_alias
    ELSE sc.child_room_alias || ROW_NUMBER() OVER (PARTITION BY sc.parent_room_alias, sc.child_room_alias ORDER BY sc.parent_room_alias, sc.origin_server_ts)
    END as child_room_alias
    FROM sc
)
    SELECT rooms.room_id, CASE WHEN sc.child_room_id IS NULL THEN substring(split_part(er.json::jsonb->'content'->>'alias', ':', 1) FROM 2) ELSE substring(split_part(sc.parent_room_alias, ':', 1) FROM 2)::text || '/' || (sc.child_room_alias) END as room_alias
    FROM rooms
    LEFT JOIN event_json er ON er.room_id = rooms.room_id AND er.json::jsonb->>'type' = 'm.room.canonical_alias'
    LEFT JOIN space_children sc ON sc.child_room_id = rooms.room_id;


CREATE UNIQUE INDEX IF NOT EXISTS aliases_idx ON aliases (room_id);

CREATE OR REPLACE FUNCTION aliases_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY aliases;
    RETURN NULL;
END;
$$;

CREATE TRIGGER aliases_mv_trigger 
AFTER INSERT
ON current_state_events
FOR EACH ROW
WHEN (NEW.type = 'm.room.create' OR NEW.type = 'm.space.parent' OR NEW.type = 'm.space.child' OR NEW.type = 'm.room.name')
EXECUTE FUNCTION aliases_mv_refresh();
