DROP INDEX IF EXISTS spaces_idx;
DROP MATERIALIZED VIEW IF EXISTS spaces;
DROP TRIGGER spaces_mv_trigger on current_state_events;
DROP FUNCTION spaces_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS spaces AS 
    SELECT rooms.room_id, er.json::jsonb->'content'->>'alias' as room_alias, substring(split_part(er.json::jsonb->'content'->>'alias', ':', 1) FROM 2) as space_alias, 
    CASE WHEN (ejs.json::jsonb->'content'->>'default')::bool = true THEN true ELSE false END as is_default
    FROM rooms
    JOIN event_json er ON er.room_id = rooms.room_id AND er.json::jsonb->>'type' = 'm.room.canonical_alias'
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
AFTER INSERT 
ON current_state_events
FOR EACH ROW
WHEN (NEW.type = 'm.room.create' 
    OR NEW.type = 'm.space.default'
    OR NEW.type = 'm.room.canonical_alias')
EXECUTE FUNCTION spaces_mv_refresh();
