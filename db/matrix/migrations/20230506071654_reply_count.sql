-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS reply_count (
    relates_to_id text,
    count bigint
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE reply_count;
-- +goose StatementEnd
