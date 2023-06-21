DROP INDEX IF EXISTS room_state_idx;
DROP MATERIALIZED VIEW IF EXISTS room_state;
DROP TRIGGER room_state_mv_trigger on current_state_events;
DROP FUNCTION room_state_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS room_state AS 
    SELECT DISTINCT ON (rooms.room_id) rooms.room_id, ra.room_alias, substring(split_part(ra.room_alias, ':', 1) FROM 2) as alias, COALESCE(rt.type, 'chat') as type, 
    CASE WHEN st.type = 'profile' THEN true ELSE false END as is_profile, n.name, t.topic, av.avatar, h.header, pev.pinned_events,
    CASE WHEN rstr.age IS NULL AND rstr.verified IS NULL THEN NULL
    ELSE jsonb_strip_nulls(jsonb_build_object(
        'age', rstr.age::int,
        'verified', CASE WHEN rstr.verified = 'true' THEN true ELSE false END
    )) END as restrictions
    FROM rooms
    LEFT JOIN (
        SELECT ej.json::jsonb->'content'->>'type' as type, cse.room_id FROM current_state_events as cse JOIN event_json as ej ON cse.event_id = ej.event_id WHERE cse.type='m.space.child.type'
    ) as rt ON rt.room_id = rooms.room_id
    LEFT JOIN (
        SELECT ej.json::jsonb->'content'->>'type' as type, cse.room_id FROM current_state_events as cse JOIN event_json as ej ON cse.event_id = ej.event_id WHERE cse.type='m.space.type'
    ) as st ON st.room_id = rooms.room_id
    LEFT JOIN (
        SELECT ej.json::jsonb->'content'->>'name' as name, cse.room_id FROM current_state_events as cse JOIN event_json as ej ON cse.event_id = ej.event_id WHERE cse.type='m.room.name'
    ) as n ON n.room_id = rooms.room_id
    LEFT JOIN (
        SELECT ej.json::jsonb->'content'->>'topic' as topic, cse.room_id FROM current_state_events as cse JOIN event_json as ej ON cse.event_id = ej.event_id WHERE cse.type='m.room.topic'
    ) as t ON t.room_id = rooms.room_id
    LEFT JOIN (
        SELECT ej.json::jsonb->'content'->>'url' as avatar, cse.room_id FROM current_state_events as cse JOIN event_json as ej ON cse.event_id = ej.event_id WHERE cse.type='m.room.avatar'
    ) as av ON av.room_id = rooms.room_id
    LEFT JOIN (
        SELECT ej.json::jsonb->'content'->>'url' as header, cse.room_id FROM current_state_events as cse JOIN event_json as ej ON cse.event_id = ej.event_id WHERE cse.type='m.room.header'
    ) as h ON h.room_id = rooms.room_id
    LEFT JOIN (
        SELECT ej.json::jsonb->'content'->>'pinned' as pinned_events, cse.room_id FROM current_state_events as cse JOIN event_json as ej ON cse.event_id = ej.event_id WHERE cse.type='m.room.pinned_events'
    ) as pev ON pev.room_id = rooms.room_id
    LEFT JOIN (
        SELECT ej.json::jsonb->'content'->>'age' as age, ej.json::jsonb->'content'->>'verified' as verified, cse.room_id FROM current_state_events as cse JOIN event_json as ej ON cse.event_id = ej.event_id WHERE cse.type='m.restrict_events_to'
    ) as rstr ON rstr.room_id = rooms.room_id
    LEFT JOIN room_aliases ra ON ra.room_id = rooms.room_id
    GROUP BY rooms.room_id, ra.room_alias, st.type, rt.type, n.name, t.topic, av.avatar, h.header, pev.pinned_events, rstr.age, rstr.verified;

CREATE UNIQUE INDEX IF NOT EXISTS room_state_idx ON room_state (room_id);

CREATE OR REPLACE FUNCTION room_state_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY room_state;
    RETURN NULL;
END;
$$;

CREATE TRIGGER room_state_mv_trigger 
AFTER INSERT 
ON current_state_events
EXECUTE FUNCTION room_state_mv_refresh();
