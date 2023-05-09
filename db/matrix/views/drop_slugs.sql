--This drops the materialized view slugs and the index slugs_idx
DROP INDEX IF EXISTS slugs_idx;
DROP MATERIALIZED VIEW IF EXISTS slugs;
DROP FUNCTION slugs_mv_refresh() CASCADE;
DROP TRIGGER slugs_mv_trigger on events;
