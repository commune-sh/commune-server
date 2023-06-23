-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS room_state (
    room_id text,
    room_alias text,
    alias text,
    type text,
    is_profile text,
    name text,
    topic text,
    avatar text,
    header text,
    pinned_events text,
    restrictions text,
    do_not_index bool
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE room_state;
-- +goose StatementEnd
