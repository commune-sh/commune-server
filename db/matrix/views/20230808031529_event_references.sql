DROP INDEX IF EXISTS event_references_idx;
DROP MATERIALIZED VIEW IF EXISTS event_references;
DROP TRIGGER event_references_mv_trigger on event_json;
DROP FUNCTION event_references_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS event_references AS 
    SELECT ej.json::jsonb->'content'->'reference'->>'event_id' as relates_to_id, ej.event_id ,
    jsonb_build_object(
        'event_id', ej.event_id,
        'content', cast(ej.json::jsonb->>'content' as jsonb),
        'sender', jsonb_build_object(
            'id', events.sender,
            'display_name', ud.display_name,
            'avatar_url', ud.avatar_url
        )
    ) as event
    FROM event_json ej
    JOIN events ON events.event_id = ej.event_id
    LEFT JOIN membership_state ud ON ud.user_id = events.sender
    AND ud.room_id = ej.room_id
    WHERE ej.json::jsonb->'content'->'reference'->>'event_id' IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS event_references_idx ON event_references (relates_to_id);

CREATE OR REPLACE FUNCTION event_references_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY event_references;
    RETURN NULL;
END;
$$;

CREATE TRIGGER event_references_mv_trigger 
AFTER INSERT OR UPDATE 
ON event_json
FOR EACH ROW
WHEN (NEW.json::jsonb->'content'->>'reference' IS NOT NULL)
EXECUTE FUNCTION event_references_mv_refresh();

