-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS profiles (
    user_id text NOT NULL,
    displayname text,
    avatar_url text,
    full_user_id text
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE profiles;
-- +goose StatementEnd
