-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS event_relations (
    event_id text NOT NULL,
    relates_to_id text NOT NULL,
    relation_type text NOT NULL,
    aggregation_key text
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE event_relations;
-- +goose StatementEnd
