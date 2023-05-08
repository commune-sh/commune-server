-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS reactions (
    relates_to_id text,
    aggregation_key text,
    count bigint
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE reactions;
-- +goose StatementEnd
