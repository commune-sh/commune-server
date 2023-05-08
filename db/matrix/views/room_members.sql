--This creates a materialized view of all the spaces and their children
--FIX THIS LATER

DROP INDEX IF EXISTS room_members_idx;
DROP MATERIALIZED VIEW IF EXISTS room_members;

CREATE MATERIALIZED VIEW IF NOT EXISTS room_members AS 
    WITH sel AS (
    SELECT ej.room_id,
        room_aliases.room_alias,
        COUNT(CASE WHEN ej.json::jsonb->'content'->>'membership' = 'join' THEN 1 ELSE null END) as join_count,
        COUNT(CASE WHEN ej.json::jsonb->'content'->>'membership' = 'leave' THEN 1 ELSE null END) as leave_count
    FROM event_json as ej
    LEFT JOIN room_aliases ON room_aliases.room_id = ej.room_id
    GROUP BY ej.room_id, room_aliases.room_alias
    ) Select sel.room_id, sel.room_alias, COALESCE(sel.join_count - sel.leave_count) as members FROM sel;

CREATE UNIQUE INDEX IF NOT EXISTS room_members_idx ON room_members (room_id);

CREATE OR REPLACE FUNCTION room_members_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY room_members;
    RETURN NULL;
END;
$$;

CREATE TRIGGER room_members_mv_trigger 
AFTER INSERT
ON events
FOR EACH ROW
WHEN (NEW.type = 'm.room.member')
EXECUTE FUNCTION room_members_mv_refresh();
