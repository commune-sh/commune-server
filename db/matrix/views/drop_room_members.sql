--This drops the materialized view room_members and the index room_members_idx
DROP INDEX IF EXISTS room_members_idx;
DROP MATERIALIZED VIEW IF EXISTS room_members;
DROP FUNCTION room_members_mv_refresh() CASCADE;
DROP TRIGGER room_members_mv_trigger on room_memberships;
