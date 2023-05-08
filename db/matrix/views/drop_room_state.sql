--This drops the materialized view room_state and the index room_state_idx
DROP INDEX IF EXISTS room_state_idx;
DROP MATERIALIZED VIEW IF EXISTS room_state;
DROP FUNCTION room_state_mv_refresh() CASCADE;
DROP TRIGGER room_state_mv_trigger on current_state_events;
