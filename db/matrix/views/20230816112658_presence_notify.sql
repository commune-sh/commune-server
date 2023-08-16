DROP TRIGGER IF EXISTS presence_stream_trigger ON presence_stream;
DROP FUNCTION IF EXISTS presence_stream_trigger_function();


CREATE OR REPLACE FUNCTION presence_stream_trigger_function()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE notification_payload text;
BEGIN
    SELECT 
    jsonb_build_object(
        'user_id', ps.user_id,
        'state', ps.state,
        'last_active_ts', ps.last_active_ts,
        'currently_active', ps.currently_active)
    INTO notification_payload
    FROM presence_stream ps;

  PERFORM pg_notify('presence_notification', notification_payload);

  RETURN NEW;
END;
$$;

CREATE TRIGGER presence_stream_trigger
AFTER INSERT OR UPDATE ON presence_stream
FOR EACH ROW
EXECUTE FUNCTION presence_stream_trigger_function();

