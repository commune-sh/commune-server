DROP INDEX IF EXISTS room_topics_idx;
DROP MATERIALIZED VIEW IF EXISTS room_topics;
DROP TRIGGER room_topics_mv_trigger on current_state_events;
DROP FUNCTION room_topics_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS room_topics AS 
    SELECT ej.room_id, ej.json::json->'content'->>'topics' AS topics
    FROM current_state_events cse
    JOIN event_json ej ON ej.event_id = cse.event_id
    WHERE cse.type = 'm.room.topics' ;

CREATE UNIQUE INDEX IF NOT EXISTS room_topics_idx ON room_topics (room_id);

CREATE OR REPLACE FUNCTION room_topics_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY room_topics;
    RETURN NULL;
END;
$$;

CREATE TRIGGER room_topics_mv_trigger 
AFTER INSERT OR UPDATE
ON current_state_events
FOR EACH ROW
WHEN (NEW.type = 'm.room.topics')
EXECUTE FUNCTION room_topics_mv_refresh();
