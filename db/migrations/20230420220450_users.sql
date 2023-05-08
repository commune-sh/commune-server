-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    email text NOT NULL UNIQUE,
    username text NOT NULL UNIQUE,
    password text NOT NULL,
    verified boolean DEFAULT false NOT NULL,
    deactivated boolean DEFAULT false NOT NULL,
    private boolean DEFAULT false NOT NULL,
    image text,
    name text,
    about text,
    info jsonb,
    settings jsonb,
    bot boolean DEFAULT false NOT NULL,
    unlisted boolean DEFAULT false NOT NULL,
    nsfw boolean DEFAULT false NOT NULL,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz,
    suspended boolean DEFAULT false NOT NULL,
    suspended_at timestamptz,
    deactivated_at timestamptz,
    reactivated_at timestamptz,
    deleted_at timestamptz,
    deleted boolean DEFAULT false NOT NULL
);

CREATE INDEX users_username_index on users(username);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX users_username_index;
DROP TABLE users;
-- +goose StatementEnd
