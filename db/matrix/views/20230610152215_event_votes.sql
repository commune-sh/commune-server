DROP INDEX IF EXISTS event_votes_idx;
DROP MATERIALIZED VIEW IF EXISTS event_votes;
DROP TRIGGER event_votes_mv_trigger on event_relations;
DROP FUNCTION event_votes_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS event_votes AS 
    SELECT relates_to_id,
           COUNT(CASE WHEN aggregation_key = 'upvote' THEN 1 END) AS upvotes,
           COUNT(CASE WHEN aggregation_key = 'downvote' THEN 1 END) AS downvotes
    FROM event_relations
    WHERE relation_type = 'm.annotation'
    GROUP BY relates_to_id;

CREATE UNIQUE INDEX IF NOT EXISTS event_votes_idx ON event_votes (relates_to_id);

CREATE OR REPLACE FUNCTION event_votes_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY event_votes;
    RETURN NULL;

    IF (TG_OP = 'DELETE') THEN
        REFRESH MATERIALIZED VIEW CONCURRENTLY event_votes;
    ELSIF (NEW.relation_type = 'm.annotation') THEN
        REFRESH MATERIALIZED VIEW CONCURRENTLY event_votes;
    END IF;

    RETURN NULL; 
END;
$$;

CREATE TRIGGER event_votes_mv_trigger 
AFTER INSERT OR UPDATE OR DELETE
ON event_relations
FOR EACH ROW
EXECUTE FUNCTION event_votes_mv_refresh();

