-- +goose Up
-- +goose StatementBegin
CREATE TABLE notifications (
    for_matrix_user_id text REFERENCES users(name) NOT NULL,
    from_matrix_user_id text REFERENCES users(name) NOT NULL,
    relates_to_event_id text NOT NULL DEFAULT '',
    event_id text NOT NULL DEFAULT '',
    thread_event_id TEXT NOT NULL DEFAULT '',
    type text NOT NULL,
    body text NOT NULL DEFAULT '',
    room_alias text NOT NULL DEFAULT '',
    created_at timestamptz DEFAULT now(),
    read_at timestamptz,
    read boolean DEFAULT false NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE notifications;
-- +goose StatementEnd
