-- +goose Up
-- +goose StatementBegin
CREATE TABLE notifications (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    matrix_user_id text REFERENCES users(matrix_user_id) NOT NULL,
    type text NOT NULL,
    content jsonb NOT NULL,
    created_at timestamptz DEFAULT now(),
    read_at timestamptz,
    read boolean DEFAULT false NOT NULL
);

CREATE INDEX notifications_index on notifications(matrix_user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX notifications_index;
DROP TABLE notifications;
-- +goose StatementEnd
