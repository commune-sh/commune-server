-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS room_aliases (
    room_alias text NOT NULL,
    room_id text NOT NULL,
    creator text,
    slug char(7)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE room_aliases;
-- +goose StatementEnd
