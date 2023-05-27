DROP INDEX IF EXISTS aliases_idx;
DROP MATERIALIZED VIEW IF EXISTS aliases;
DROP TRIGGER aliases_mv_trigger on current_state_events;
DROP FUNCTION aliases_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS aliases AS 
    SELECT rooms.room_id, CASE WHEN sc.child_room_id IS NULL THEN substring(split_part(ra.room_alias, ':', 1) FROM 2) ELSE substring(split_part(sc.parent_room_alias, ':', 1) FROM 2)::text || '/' || (sc.child_room_alias) END as room_alias
    FROM rooms
    LEFT JOIN room_aliases ra ON ra.room_id = rooms.room_id
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
AFTER INSERT OR UPDATE OR DELETE
ON current_state_events
EXECUTE FUNCTION aliases_mv_refresh();
