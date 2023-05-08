--This creates a materialized view of all the spaces and their children
--FIX THIS LATER

DROP INDEX IF EXISTS user_reactions_idx;
DROP MATERIALIZED VIEW IF EXISTS user_reactions;

CREATE MATERIALIZED VIEW IF NOT EXISTS user_reactions AS 
    SELECT er.relates_to_id, ev.sender, array_agg(er.aggregation_key) as reactions
    FROM event_relations er 
    LEFT JOIN events ev ON ev.event_id = er.event_id
    WHERE relation_type = 'm.annotation' 
    GROUP BY er.relates_to_id, ev.sender;

CREATE UNIQUE INDEX IF NOT EXISTS user_reactions_idx ON user_reactions (relates_to_id, sender);

CREATE OR REPLACE FUNCTION user_reactions_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_reactions;
    RETURN NULL;
END;
$$;

CREATE TRIGGER user_reactions_mv_trigger 
AFTER INSERT 
ON event_relations
FOR EACH ROW
WHEN (NEW.relation_type = 'm.annotation')
EXECUTE FUNCTION user_reactions_mv_refresh();
