--This creates a materialized view of all the spaces and their children
--FIX THIS LATER

DROP INDEX IF EXISTS reactions_idx;
DROP MATERIALIZED VIEW IF EXISTS reactions;

CREATE MATERIALIZED VIEW IF NOT EXISTS reactions AS 
    SELECT er.relates_to_id, er.aggregation_key, count(*) 
    FROM event_relations er 
    WHERE relation_type = 'm.annotation' 
    GROUP BY aggregation_key, relates_to_id;

CREATE UNIQUE INDEX IF NOT EXISTS reactions_idx ON reactions (relates_to_id, aggregation_key);

CREATE OR REPLACE FUNCTION reactions_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY reactions;
    RETURN NULL;
END;
$$;

CREATE TRIGGER reactions_mv_trigger 
AFTER INSERT 
ON event_relations
FOR EACH ROW
WHEN (NEW.relation_type = 'm.annotation')
EXECUTE FUNCTION reactions_mv_refresh();
