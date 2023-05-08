-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_reactions (
    relates_to_id text,
    sender text,
    reactions text[]
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_reactions;
-- +goose StatementEnd
