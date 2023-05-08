-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS event_reactions (
    relates_to_id text,
    aggregation_key text,
    senders text[]
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE event_reactions;
-- +goose StatementEnd
