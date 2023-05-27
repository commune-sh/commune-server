DROP INDEX IF EXISTS reply_count_idx;
DROP MATERIALIZED VIEW IF EXISTS reply_count;
DROP TRIGGER reply_count_mv_trigger on event_relations;
DROP FUNCTION reply_count_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS reply_count AS 
    SELECT er.relates_to_id, count(*) 
    FROM event_relations er 
    WHERE relation_type = 'm.thread' 
    GROUP BY aggregation_key, relates_to_id;

CREATE UNIQUE INDEX IF NOT EXISTS reply_count_idx ON reply_count (relates_to_id, count);

CREATE OR REPLACE FUNCTION reply_count_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY reply_count;
    RETURN NULL;
END;
$$;

CREATE TRIGGER reply_count_mv_trigger 
AFTER INSERT 
ON event_relations
FOR EACH ROW
WHEN (NEW.relation_type = 'm.thread')
EXECUTE FUNCTION reply_count_mv_refresh();
