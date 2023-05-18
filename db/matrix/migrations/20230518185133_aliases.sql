-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS aliases (
    room_id text NOT NULL,
    room_alias text NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE aliases;
-- +goose StatementEnd
