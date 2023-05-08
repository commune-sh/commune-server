--This drops the materialized view space_children and the index space_children_idx
DROP INDEX IF EXISTS space_children_idx;
DROP MATERIALIZED VIEW IF EXISTS space_children;
DROP FUNCTION space_children_mv_refresh() CASCADE;
DROP TRIGGER space_children_mv_trigger on current_state_events;
