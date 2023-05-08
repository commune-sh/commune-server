--This drops the materialized view reply_count and the index reply_count_idx
DROP INDEX IF EXISTS reply_count_idx;
DROP MATERIALIZED VIEW IF EXISTS reply_count;
DROP FUNCTION reply_count_mv_refresh() CASCADE;
DROP TRIGGER reply_count_mv_trigger on event_relations;
