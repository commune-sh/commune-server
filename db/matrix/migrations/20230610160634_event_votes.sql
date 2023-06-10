-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS event_votes (
    relates_to_id text,
    upvotes bigint,
    downvotes bigint
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE event_votes;
-- +goose StatementEnd
