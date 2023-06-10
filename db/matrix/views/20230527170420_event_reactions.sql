DROP INDEX IF EXISTS event_reactions_idx;
DROP MATERIALIZED VIEW IF EXISTS event_reactions;
DROP TRIGGER event_reactions_mv_trigger on event_relations;
DROP FUNCTION event_reactions_mv_refresh();

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

    IF (TG_OP = 'DELETE') THEN
        REFRESH MATERIALIZED VIEW CONCURRENTLY event_reactions;
    ELSIF (NEW.relation_type = 'm.annotation') THEN
        REFRESH MATERIALIZED VIEW CONCURRENTLY event_reactions;
    END IF;

    RETURN NULL; 
END;
$$;

CREATE TRIGGER event_reactions_mv_trigger 
AFTER INSERT OR UPDATE OR DELETE
ON event_relations
FOR EACH ROW
EXECUTE FUNCTION event_reactions_mv_refresh();
