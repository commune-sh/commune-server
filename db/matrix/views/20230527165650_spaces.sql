DROP INDEX IF EXISTS spaces_idx;
DROP MATERIALIZED VIEW IF EXISTS spaces;
DROP TRIGGER spaces_mv_trigger on current_state_events;
DROP FUNCTION spaces_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS spaces AS 
    SELECT rooms.room_id, ra.room_alias, substring(split_part(LOWER(ra.room_alias), ':', 1) FROM 2) as space_alias, 
    CASE WHEN (ejs.json::jsonb->'content'->>'default')::bool = true THEN true ELSE false END as default
    FROM rooms
    JOIN room_aliases ra ON ra.room_id = rooms.room_id
    JOIN event_json ej ON ej.room_id = rooms.room_id AND ej.json::jsonb->>'type' = 'm.room.create' AND ej.json::jsonb->'content'->>'type' = 'm.space'
    LEFT JOIN current_state_events cse ON cse.room_id = rooms.room_id AND cse.type = 'm.space.default'
    LEFT JOIN event_json ejs ON ejs.event_id = cse.event_id;

CREATE UNIQUE INDEX IF NOT EXISTS spaces_idx ON spaces (room_id);

CREATE OR REPLACE FUNCTION spaces_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY spaces;
    RETURN NULL;
END;
$$;

CREATE TRIGGER spaces_mv_trigger 
AFTER INSERT OR UPDATE OR DELETE
ON current_state_events
EXECUTE FUNCTION spaces_mv_refresh();
