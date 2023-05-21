--This drops the materialized view spaces and the index spaces_idx
DROP INDEX IF EXISTS spaces_idx;
DROP MATERIALIZED VIEW IF EXISTS spaces;
DROP FUNCTION spaces_mv_refresh() CASCADE;
DROP TRIGGER spaces_mv_trigger on current_state_events;
