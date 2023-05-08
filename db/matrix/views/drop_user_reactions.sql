--This drops the materialized view user_reactions and the index user_reactions_idx
DROP INDEX IF EXISTS user_reactions_idx;
DROP MATERIALIZED VIEW IF EXISTS user_reactions;
DROP FUNCTION user_reactions_mv_refresh() CASCADE;
DROP TRIGGER user_reactions_mv_trigger on event_relations;
