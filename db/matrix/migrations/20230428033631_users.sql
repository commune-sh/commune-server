-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    name text,
    password_hash text,
    creation_ts bigint,
    admin smallint NOT NULL DEFAULT 0,
    upgrade_ts bigint,
    is_guest smallint NOT NULL DEFAULT 0,
    appservice_id text,
    consent_version text,
    consent_server_notice_sent text,
    user_type text,
    deactivated smallint NOT NULL DEFAULT 0,
    shadow_banned boolean,
    consent_ts bigint,
    approved boolean
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
