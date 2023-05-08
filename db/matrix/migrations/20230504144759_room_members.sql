-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS room_members (
    room_id text,
    room_alias text,
    members bigint
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE room_members;
-- +goose StatementEnd
