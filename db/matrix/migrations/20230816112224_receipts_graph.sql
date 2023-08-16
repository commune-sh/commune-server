-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS receipts_graph (
    room_id text NOT NULL,
    receipt_type text NOT NULL,
    user_id text NOT NULL,
    event_ids text NOT NULL,
    data text NOT NULL,
    thread_id text
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE receipts_graph;
-- +goose StatementEnd
