-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS pinned_events (
    room_id text,
    events text[]
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE pinned_events;
-- +goose StatementEnd
