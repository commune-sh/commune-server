DROP INDEX IF EXISTS pinned_events_idx;
DROP MATERIALIZED VIEW IF EXISTS pinned_events;
DROP TRIGGER pinned_events_mv_trigger on current_state_events;
DROP FUNCTION pinned_events_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS pinned_events AS 
    SELECT ej.room_id, ej.json::jsonb->'content'->>'pinned' as events
    FROM current_state_events cse
    JOIN event_json ej ON ej.event_id = cse.event_id
    WHERE cse.type = 'm.room.pinned_events';

CREATE UNIQUE INDEX IF NOT EXISTS pinned_events_idx ON pinned_events (room_id);

CREATE OR REPLACE FUNCTION pinned_events_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY pinned_events;
    RETURN NULL;
END;
$$;

CREATE TRIGGER pinned_events_mv_trigger 
AFTER INSERT OR UPDATE OR DELETE
ON current_state_events
EXECUTE FUNCTION pinned_events_mv_refresh();
