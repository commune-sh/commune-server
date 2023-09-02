-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_external_ids (
    auth_provider text not null,
    external_id text not null,
    user_id text not null
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_external_ids;
-- +goose StatementEnd
