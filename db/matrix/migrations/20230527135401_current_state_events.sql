-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS current_state_events (
    event_id text NOT NULL,
    room_id text NOT NULL,
    type text NOT NULL,
    state_key text NOT NULL,
    membership text,
    event_stream_ordering bigint
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE current_state_events;
-- +goose StatementEnd
