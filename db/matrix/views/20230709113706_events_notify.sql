DROP TRIGGER IF EXISTS events_insert_trigger ON events;
DROP FUNCTION IF EXISTS events_trigger_function();


CREATE OR REPLACE FUNCTION events_trigger_function()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE notification_payload text;
BEGIN
    SELECT 
    jsonb_build_object(
        'event_id', ej.event_id, 
        'type', ev.type,
        'room_id', ev.room_id)
    INTO notification_payload
    FROM event_json ej
    JOIN events ev ON ej.event_id = ev.event_id
    WHERE ev.event_id = NEW.event_id;

  PERFORM pg_notify('events_notification', notification_payload);

  RETURN NEW;
END;
$$;

CREATE TRIGGER events_insert_trigger
AFTER INSERT ON events
FOR EACH ROW
WHEN (NEW.type = 'm.room.message' 
    OR NEW.type = 'm.reaction'
    OR NEW.type = 'm.room.member'
    OR NEW.type = 'm.room.redaction'
    OR NEW.type = 'm.room.name'
    OR NEW.type = 'm.room.topic'
    OR NEW.type = 'space.board.post'
    OR NEW.type = 'space.board.post.reply')
EXECUTE FUNCTION events_trigger_function();

