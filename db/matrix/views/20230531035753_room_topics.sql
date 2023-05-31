DROP INDEX IF EXISTS room_topics_idx;
DROP MATERIALIZED VIEW IF EXISTS room_topics;
DROP TRIGGER room_topics_mv_trigger on current_state_events;
DROP FUNCTION room_topics_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS room_topics AS 
    SELECT DISTINCT ON (cse.room_id) cse.room_id, COALESCE(array_agg(DISTINCT cse.state_key), null) as topics
    FROM current_state_events cse 
    WHERE cse.type = 'm.room.stream'
    AND cse.state_key != ''
    GROUP BY cse.room_id;

CREATE UNIQUE INDEX IF NOT EXISTS room_topics_idx ON room_topics (room_id);

CREATE OR REPLACE FUNCTION room_topics_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY room_topics;
    RETURN NULL;
END;
$$;

CREATE TRIGGER room_topics_mv_trigger 
AFTER INSERT OR UPDATE OR DELETE
ON current_state_events
EXECUTE FUNCTION room_topics_mv_refresh();
