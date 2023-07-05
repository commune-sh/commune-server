-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE notifications (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    for_matrix_user_id text REFERENCES users(name) NOT NULL,
    from_matrix_user_id text REFERENCES users(name) NOT NULL,
    room_id text NOT NULL DEFAULT '',
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

CREATE INDEX notifications_index on notifications(for_matrix_user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX notifications_index;
DROP TABLE notifications;
-- +goose StatementEnd
