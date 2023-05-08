--This creates a materialized view of all the spaces and their children
--FIX THIS LATER

DROP INDEX IF EXISTS space_children_idx;
DROP MATERIALIZED VIEW IF EXISTS space_children;

CREATE MATERIALIZED VIEW IF NOT EXISTS space_children AS 
    SELECT ra.room_alias as parent_room_alias, ra.room_id as parent_room_id, cse.state_key as child_room_id, ra2.room_alias as child_room_alias
    FROM room_aliases ra
    LEFT JOIN current_state_events as cse ON cse.room_id = ra.room_id AND cse.type ='m.space.child'
    LEFT JOIN room_aliases ra2 ON ra2.room_id = cse.state_key
    LEFT JOIN event_json ev ON ev.event_id = cse.event_id
    WHERE ev.json::jsonb->'content'->>'via' is not null;

CREATE UNIQUE INDEX IF NOT EXISTS space_children_idx ON space_children (child_room_id);

CREATE OR REPLACE FUNCTION space_children_mv_refresh()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY space_children;
    RETURN NULL;
END;
$$;

CREATE TRIGGER space_children_mv_trigger 
AFTER INSERT OR UPDATE OR DELETE
ON current_state_events
EXECUTE FUNCTION space_children_mv_refresh();
