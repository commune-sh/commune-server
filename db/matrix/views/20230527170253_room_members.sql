DROP INDEX IF EXISTS room_members_idx;
DROP MATERIALIZED VIEW IF EXISTS room_members;
DROP TRIGGER room_members_mv_trigger on room_memberships;
DROP FUNCTION room_members_mv_refresh();

DROP INDEX IF EXISTS room_members_idx;
DROP MATERIALIZED VIEW IF EXISTS room_members;

CREATE MATERIALIZED VIEW IF NOT EXISTS room_members AS 
    WITH sel AS (
        SELECT DISTINCT ON (rm.user_id, rm.room_id) rm.user_id, rm.room_id, membership
        FROM room_memberships rm  
        JOIN events ev ON ev.event_id = rm.event_id
        GROUP BY rm.user_id, rm.room_id, rm.membership, ev.origin_server_ts
        ORDER BY rm.user_id, rm.room_id, ev.origin_server_ts DESC
    ) SELECT DISTINCT ON (sel.room_id) sel.room_id, ra.room_alias, COUNT(CASE WHEN sel.membership = 'join' THEN 1 END) as members
    FROM sel
    JOIN room_aliases ra ON ra.room_id = sel.room_id
    GROUP BY sel.room_id, ra.room_alias;

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
ON room_memberships
EXECUTE FUNCTION room_members_mv_refresh();
