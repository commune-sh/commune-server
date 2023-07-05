-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_threepids (
    user_id text NOT NULL,
    medium text NOT NULL,
    address text NOT NULL,
    validated_at bigint NOT NULL,
    added_at bigint NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_threepids;
-- +goose StatementEnd
