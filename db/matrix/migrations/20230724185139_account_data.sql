-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS account_data (
    user_id text not null,
    account_data_type text not null,
    stream_id bigint not null,
    content text not null,
    instance_name text
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE account_data;
-- +goose StatementEnd
