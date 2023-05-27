DROP INDEX IF EXISTS membership_state_idx;
DROP MATERIALIZED VIEW IF EXISTS membership_state;
DROP TRIGGER membership_state_mv_trigger on room_memberships;
DROP FUNCTION membership_state_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS membership_state AS 
	SELECT DISTINCT ON (rm.room_id, user_id) rm.user_id, rm.room_id, membership
	FROM room_memberships rm  
	JOIN events ev ON ev.event_id = rm.event_id
	GROUP BY rm.user_id, rm.room_id, rm.membership, ev.origin_server_ts
	ORDER BY rm.room_id, rm.user_id, ev.origin_server_ts DESC;

CREATE UNIQUE INDEX IF NOT EXISTS membership_state_idx ON membership_state (user_id, room_id);

CREATE OR REPLACE FUNCTION membership_state_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY membership_state;
    RETURN NULL;
END;
$$;

CREATE TRIGGER membership_state_mv_trigger 
AFTER INSERT
ON room_memberships
EXECUTE FUNCTION membership_state_mv_refresh();
