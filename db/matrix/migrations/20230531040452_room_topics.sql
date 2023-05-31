-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS room_topics (
    room_id text,
    topics text[]
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE room_topics;
-- +goose StatementEnd
