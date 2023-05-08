-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_directory (
    user_id text NOT NULL,
    room_id text,
    display_name text,
    avatar_url text
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_directory;
-- +goose StatementEnd
