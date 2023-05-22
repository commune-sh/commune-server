--This creates a materialized view of all the spaces and their children
--FIX THIS LATER

DROP INDEX IF EXISTS aliases_idx;
DROP MATERIALIZED VIEW IF EXISTS aliases;

CREATE MATERIALIZED VIEW IF NOT EXISTS aliases AS 
    SELECT rooms.room_id, CASE WHEN sc.child_room_id IS NULL THEN substring(split_part(ra.room_alias, ':', 1) FROM 2) ELSE substring(split_part(sc.parent_room_alias, ':', 1) FROM 2)::text || '/' || (sc.child_room_alias) END as room_alias
    FROM rooms
    LEFT JOIN room_aliases ra ON ra.room_id = rooms.room_id
    LEFT JOIN (
        SELECT ra.room_alias as parent_room_alias, ra.room_id as parent_room_id, cse.state_key as child_room_id, sch.alias as child_room_alias
        FROM room_aliases ra
        LEFT JOIN current_state_events as cse ON cse.room_id = ra.room_id AND cse.type ='m.space.child'
        LEFT JOIN event_json ev ON ev.event_id = cse.event_id
        LEFT JOIN (
        SELECT cs.room_id, ej.json::jsonb->'content'->>'name' as alias FROM current_state_events cs 
        JOIN event_json ej ON ej.event_id = cs.event_id
        WHERE cs.type = 'm.room.name'
        ) as sch ON sch.room_id = cse.state_key
        WHERE ev.json::jsonb->'content'->>'via' is not null
    ) as sc ON sc.child_room_id = rooms.room_id;

CREATE UNIQUE INDEX IF NOT EXISTS aliases_idx ON aliases (room_id);

CREATE OR REPLACE FUNCTION aliases_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY aliases;
    RETURN NULL;
END;
$$;

CREATE TRIGGER aliases_mv_trigger 
AFTER INSERT OR UPDATE OR DELETE
ON current_state_events
EXECUTE FUNCTION aliases_mv_refresh();
