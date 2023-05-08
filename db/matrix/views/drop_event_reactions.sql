--This drops the materialized view event_reactions and the index event_reactions_idx
DROP INDEX IF EXISTS event_reactions_idx;
DROP MATERIALIZED VIEW IF EXISTS event_reactions;
DROP FUNCTION event_reactions_mv_refresh() CASCADE;
DROP TRIGGER event_reactions_mv_trigger on event_relations;
