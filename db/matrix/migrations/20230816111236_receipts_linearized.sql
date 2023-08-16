-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS receipts_linearized (
    stream_id bigint NOT NULL,
    room_id text NOT NULL,
    receipt_type text NOT NULL,
    user_id text NOT NULL,
    event_id text NOT NULL,
    data text NOT NULL,
    instance_name text,
    event_stream_ordering bigint,
    thread_id text
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE receipts_linearized;
-- +goose StatementEnd
