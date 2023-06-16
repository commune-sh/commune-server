-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS event_search (
    event_id text,
    room_id text,
    sender text,
    key text,
    vector tsvector,
    origin_server_ts bigint,
    stream_ordering bigint
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE event_search;
-- +goose StatementEnd
