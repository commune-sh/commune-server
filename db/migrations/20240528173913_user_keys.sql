-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_keys (
    matrix_user_id text NOT NULL UNIQUE,
    public_key BYTEA NOT NULL,
    private_key TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_keys;
-- +goose StatementEnd
