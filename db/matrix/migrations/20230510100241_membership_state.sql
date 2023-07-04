-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS membership_state (
    user_id text,
    room_id text,
    membership text,
    display_name text,
    avatar_url text
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE membership_state;
-- +goose StatementEnd
