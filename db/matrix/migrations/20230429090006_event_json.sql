-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS event_json (
    event_id text NOT NULL,
    room_id text NOT NULL,
    internal_metadata text NOT NULL,
    json text,
    format_version integer
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE event_json;
-- +goose StatementEnd
