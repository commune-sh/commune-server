-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS event_txn_id_device_id (
    event_id text not null,
    room_id text not null,
    user_id text not null,
    device_id bigint not null,
    txn_id text not null,
    inserted_ts bigint not null
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE event_txn_id_device_id;
-- +goose StatementEnd
