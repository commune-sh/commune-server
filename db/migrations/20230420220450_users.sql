-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    matrix_user_id text NOT NULL UNIQUE,
    email text UNIQUE,
    username text NOT NULL UNIQUE,
    verified boolean DEFAULT false NOT NULL,
    created_at timestamptz DEFAULT now(),
    deleted boolean DEFAULT false NOT NULL,
    deleted_at timestamptz
);

CREATE INDEX users_id_index on users(id);
CREATE INDEX users_matrix_user_id_index on users(matrix_user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX users_id_index;
DROP INDEX users_matrix_user_id_index;
DROP TABLE users;
-- +goose StatementEnd
