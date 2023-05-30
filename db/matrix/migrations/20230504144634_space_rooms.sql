-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS space_rooms (
    parent_room_alias text,
    parent_room_id text,
    child_room_alias text,
    child_room_id text
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE space_rooms;
-- +goose StatementEnd
