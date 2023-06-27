DROP INDEX IF EXISTS space_rooms_idx;
DROP MATERIALIZED VIEW IF EXISTS space_rooms CASCADE;
DROP TRIGGER space_rooms_mv_trigger on current_state_events;
DROP FUNCTION space_rooms_mv_refresh();

CREATE MATERIALIZED VIEW IF NOT EXISTS space_rooms AS 
    WITH sc AS (
        SELECT ra.room_alias as parent_room_alias, ra.room_id as parent_room_id, cse.state_key as child_room_id, trim(BOTH '-' FROM regexp_replace(lower(unaccent(trim(sc.alias))), '[^a-z0-9\\_-]+', '-', 'gi')) as child_room_alias, events.origin_server_ts
        FROM room_aliases ra
        LEFT JOIN current_state_events as cse ON cse.room_id = ra.room_id AND cse.type ='m.space.child'
        LEFT JOIN event_json ev ON ev.event_id = cse.event_id
        JOIN events ON events.event_id = ev.event_id
        LEFT JOIN (
        SELECT cs.room_id, COALESCE(ej.json::jsonb->'content'->>'name'::text, 'untitled') as alias FROM current_state_events cs 
        JOIN event_json ej ON ej.event_id = cs.event_id
        WHERE cs.type = 'm.room.name'
        ) as sc ON sc.room_id = cse.state_key
        WHERE ev.json::jsonb->'content'->>'via' is not null
        ORDER BY events.origin_server_ts DESC
    ) SELECT sc.parent_room_alias, sc.parent_room_id, sc.child_room_id,
    CASE WHEN ROW_NUMBER() OVER (PARTITION BY sc.parent_room_alias, sc.child_room_alias ORDER BY sc.parent_room_alias, sc.origin_server_ts) = 1 THEN sc.child_room_alias
    ELSE sc.child_room_alias || ROW_NUMBER() OVER (PARTITION BY sc.parent_room_alias, sc.child_room_alias ORDER BY sc.parent_room_alias, sc.origin_server_ts)
    END as child_room_alias
    FROM sc;

CREATE UNIQUE INDEX IF NOT EXISTS space_rooms_idx ON space_rooms (child_room_id);

CREATE OR REPLACE FUNCTION space_rooms_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY space_rooms;
    RETURN NULL;
END;
$$;

CREATE TRIGGER space_rooms_mv_trigger 
AFTER INSERT
ON current_state_events
FOR EACH ROW
WHEN (NEW.type = 'm.space.child')
EXECUTE FUNCTION space_rooms_mv_refresh();
