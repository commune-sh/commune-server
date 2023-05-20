-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS room_state (
    room_id text,
    room_alias text,
    type text,
    name text,
    topic text,
    avatar text,
    header text
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE room_state;
-- +goose StatementEnd
