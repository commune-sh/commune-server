DROP INDEX IF EXISTS event_activity_idx;
DROP MATERIALIZED VIEW IF EXISTS event_activity;
DROP TRIGGER event_activity_mv_trigger on events;
DROP FUNCTION event_activity_mv_refresh();

/*
CREATE MATERIALIZED VIEW IF NOT EXISTS event_activity AS 
    SELECT events.event_id, 
    CASE WHEN lu.origin_server_ts IS NULL THEN events.origin_server_ts 
    ELSE lu.origin_server_ts END
    FROM events
    LEFT JOIN (
        SELECT events.origin_server_ts, er.relates_to_id
        FROM events
        JOIN event_relations er 
        ON er.event_id = events.event_id
        WHERE events.type = 'space.board.post.reply'
        OR events.type = 'm.reaction'
        ORDER BY events.origin_server_ts DESC LIMIT 1
    ) as lu ON lu.relates_to_id = events.event_id
    WHERE events.type = 'space.board.post';

CREATE UNIQUE INDEX IF NOT EXISTS event_activity_idx ON event_activity (event_id);

CREATE OR REPLACE FUNCTION event_activity_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY event_activity;
    RETURN NULL;
END;
$$;

CREATE TRIGGER event_activity_mv_trigger 
AFTER INSERT
ON events
FOR EACH ROW
WHEN (NEW.type = 'space.board.post.reply')
EXECUTE FUNCTION event_activity_mv_refresh();
*/
