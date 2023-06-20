DROP INDEX IF EXISTS reply_count_idx;
DROP MATERIALIZED VIEW IF EXISTS reply_count;
DROP TRIGGER reply_count_mv_trigger on event_relations;
DROP FUNCTION reply_count_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS reply_count AS 
    WITH RECURSIVE recursive_events AS (
        SELECT er.event_id, er.relates_to_id, 1 AS reply_count
        FROM event_relations er
	LEFT JOIN redactions ON redactions.redacts = er.event_id
        WHERE er.relation_type = 'm.nested_reply'
	AND redactions.redacts is null

        UNION

        SELECT er.event_id, er.relates_to_id, re.reply_count + 1
        FROM event_relations er
        INNER JOIN recursive_events re ON er.event_id = re.relates_to_id
    )
    SELECT relates_to_id, COUNT(event_id) AS count
    FROM recursive_events
    GROUP BY relates_to_id;

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
WHEN (NEW.relation_type = 'm.nested_reply')
EXECUTE FUNCTION reply_count_mv_refresh();
