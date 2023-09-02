-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS devices (
    user_id text NOT NULL,
    device_id text NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE devices;
-- +goose StatementEnd
