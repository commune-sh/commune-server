DROP INDEX IF EXISTS search_idx;
DROP MATERIALIZED VIEW IF EXISTS search;
DROP TRIGGER search_mv_trigger on events;
DROP FUNCTION search_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS search AS 
    SELECT ej.event_id, 
    to_tsvector('english', ej.json::jsonb->'content'->>'title') AS title_vec,
    to_tsvector('english', ej.json::jsonb->'content'->>'body') AS body_vec
    FROM event_json ej
    JOIN events ON events.event_id = ej.event_id
    WHERE events.type = 'm.room.message';

CREATE UNIQUE INDEX IF NOT EXISTS search_idx ON search (event_id);

CREATE OR REPLACE FUNCTION search_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY search;
    RETURN NULL;
END;
$$;

CREATE TRIGGER search_mv_trigger 
AFTER INSERT 
ON events
FOR EACH ROW
WHEN (NEW.type = 'm.room.message')
EXECUTE FUNCTION search_mv_refresh();
