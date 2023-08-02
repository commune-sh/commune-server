DROP INDEX IF EXISTS event_threads_idx;
DROP MATERIALIZED VIEW IF EXISTS event_threads;
DROP TRIGGER event_threads_mv_trigger on event_relations;
DROP FUNCTION event_threads_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS event_threads AS 
    SELECT DISTINCT ON (events.event_id) events.event_id, count(er.relates_to_id) as replies, last.last_reply
    FROM events
    JOIN event_relations er ON er.relates_to_id = events.event_id
    LEFT JOIN (
        SELECT evr.relates_to_id,
        jsonb_build_object(
            'event_id', ej.event_id,
            'content', ej.json::jsonb->>'content',
            'sender', jsonb_build_object(
                'id', events.sender,
                'display_name', ud.display_name,
                'avatar_url', ud.avatar_url
            )
        ) as last_reply
        FROM event_json ej
        JOIN event_relations evr ON evr.event_id = ej.event_id
        JOIN events ON events.event_id = ej.event_id
        LEFT JOIN membership_state ud ON ud.user_id = events.sender
        AND ud.room_id = ej.room_id
        WHERE evr.relation_type = 'm.thread'
        ORDER BY events.origin_server_ts DESC
    ) as last ON last.relates_to_id = events.event_id
    WHERE er.relation_type = 'm.thread'
    GROUP BY events.event_id, last.last_reply;

CREATE UNIQUE INDEX IF NOT EXISTS event_threads_idx ON event_threads (event_id);

CREATE OR REPLACE FUNCTION event_threads_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY event_threads;
    RETURN NULL;
END;
$$;

CREATE TRIGGER event_threads_mv_trigger 
AFTER INSERT OR UPDATE OR DELETE
ON event_relations
FOR EACH ROW
EXECUTE FUNCTION event_threads_mv_refresh();

