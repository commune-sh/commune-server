-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS spaces (
    room_id text,
    room_alias text,
    is_default boolean,
    space_alias text
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE spaces;
-- +goose StatementEnd
