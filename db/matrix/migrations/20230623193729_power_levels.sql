-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS power_levels (
    room_id text,
    users jsonb,
    power_levels jsonb
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE power_levels;
-- +goose StatementEnd
