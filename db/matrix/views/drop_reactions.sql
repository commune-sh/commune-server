--This drops the materialized view reactions and the index reactions_idx
DROP INDEX IF EXISTS reactions_idx;
DROP MATERIALIZED VIEW IF EXISTS reactions;
DROP FUNCTION reactions_mv_refresh() CASCADE;
DROP TRIGGER reactions_mv_trigger on event_relations;
