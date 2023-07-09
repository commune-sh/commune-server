DROP TRIGGER IF EXISTS events_insert_trigger ON events;
DROP FUNCTION IF EXISTS events_trigger_function();


CREATE OR REPLACE FUNCTION events_trigger_function()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE notification_payload text;
BEGIN
    SELECT ej.event_id
    INTO notification_payload
    FROM event_json ej
    JOIN events e ON ej.event_id = e.event_id
    WHERE e.event_id = NEW.event_id;

  PERFORM pg_notify('events_notification', notification_payload);

  RETURN NEW;
END;
$$;

CREATE TRIGGER events_insert_trigger
AFTER INSERT ON events
FOR EACH ROW
WHEN (NEW.type = 'm.room.message')
EXECUTE FUNCTION events_trigger_function();

