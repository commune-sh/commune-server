--This creates a materialized view of all the spaces and their children
--FIX THIS LATER

DROP INDEX IF EXISTS event_reactions_idx;
DROP MATERIALIZED VIEW IF EXISTS event_reactions;

CREATE MATERIALIZED VIEW IF NOT EXISTS event_reactions AS 
    SELECT er.relates_to_id, er.aggregation_key, array_agg(ev.sender) as senders
    FROM event_relations er 
    JOIN events ev ON ev.event_id = er.event_id AND er.relation_type = 'm.annotation'
    GROUP BY er.aggregation_key, er.relates_to_id;

CREATE UNIQUE INDEX IF NOT EXISTS event_reactions_idx ON event_reactions (relates_to_id, aggregation_key);

CREATE OR REPLACE FUNCTION event_reactions_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY event_reactions;
    RETURN NULL;
END;
$$;

CREATE TRIGGER event_reactions_mv_trigger 
AFTER INSERT 
ON event_relations
FOR EACH ROW
WHEN (NEW.relation_type = 'm.annotation')
EXECUTE FUNCTION event_reactions_mv_refresh();
