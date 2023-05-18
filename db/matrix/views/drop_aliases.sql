--This drops the materialized view aliases and the index aliases_idx
DROP INDEX IF EXISTS aliases_idx;
DROP MATERIALIZED VIEW IF EXISTS aliases;
DROP FUNCTION aliases_mv_refresh() CASCADE;
DROP TRIGGER aliases_mv_trigger on current_state_events;
