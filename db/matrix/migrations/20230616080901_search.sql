-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS search (
    event_id text,
    title_vec tsvector,
    body_vec tsvector
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE search;
-- +goose StatementEnd
