DROP INDEX IF EXISTS room_streams_idx;
DROP MATERIALIZED VIEW IF EXISTS room_streams;
DROP TRIGGER room_streams_mv_trigger on current_state_events;
DROP FUNCTION room_streams_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS room_streams AS 
    SELECT DISTINCT ON (cse.room_id) cse.room_id, COALESCE(array_agg(DISTINCT cse.state_key), null) as streams
    FROM current_state_events cse 
    WHERE cse.type = 'm.room.stream'
    AND cse.state_key != ''
    GROUP BY cse.room_id;

CREATE UNIQUE INDEX IF NOT EXISTS room_streams_idx ON room_streams (room_id);

CREATE OR REPLACE FUNCTION room_streams_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY room_streams;
    RETURN NULL;
END;
$$;

CREATE TRIGGER room_streams_mv_trigger 
AFTER INSERT OR UPDATE OR DELETE
ON current_state_events
EXECUTE FUNCTION room_streams_mv_refresh();
