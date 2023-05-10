--This drops the materialized view membership_state and the index
--membership_state_idx
DROP INDEX IF EXISTS membership_state_idx;
DROP MATERIALIZED VIEW IF EXISTS membership_state;
DROP FUNCTION membership_state_mv_refresh() CASCADE;
DROP TRIGGER membership_state_mv_trigger on room_memberships;
