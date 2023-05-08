-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS room_memberships (
    event_id text not null,
    user_id text not null,
    sender text not null,
    room_id text not null,
    membership text not null,
    forgotten integer default 0,
    display_name text,
    avatar_url text
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE room_memberships;
-- +goose StatementEnd
