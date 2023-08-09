DROP INDEX IF EXISTS power_levels_idx;
DROP MATERIALIZED VIEW IF EXISTS power_levels;
DROP TRIGGER power_levels_mv_trigger on current_state_events;
DROP FUNCTION power_levels_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS power_levels AS 
    SELECT cse.room_id, cast(ej.json::jsonb->'content'->>'users' as jsonb) as users,
    cast(ej.json::jsonb->>'content' as jsonb) as power_levels
    FROM current_state_events cse
    JOIN event_json ej ON ej.event_id = cse.event_id
    WHERE cse.type = 'm.room.power_levels';

CREATE UNIQUE INDEX IF NOT EXISTS power_levels_idx ON power_levels (room_id);

CREATE OR REPLACE FUNCTION power_levels_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY power_levels;
    RETURN NULL;
END;
$$;

CREATE TRIGGER power_levels_mv_trigger 
AFTER INSERT OR UPDATE
ON current_state_events
FOR EACH ROW
WHEN (NEW.type = 'm.room.power_levels')
EXECUTE FUNCTION power_levels_mv_refresh();
