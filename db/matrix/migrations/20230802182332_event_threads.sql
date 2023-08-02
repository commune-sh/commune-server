-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS event_threads (
    event_id text,
    replies bigint,
    last_reply jsonb
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE event_threads;
-- +goose StatementEnd
