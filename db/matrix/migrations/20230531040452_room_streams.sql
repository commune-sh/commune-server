-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS room_streams (
    room_id text,
    streams text[]
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE room_streams;
-- +goose StatementEnd
