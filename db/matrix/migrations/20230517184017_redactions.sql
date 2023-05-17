-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS redactions (
    event_id text NOT NULL,
    redacts text NOT NULL,
    have_censored boolean NOT NULL DEFAULT false,
    received_ts bigint NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE redactions;
-- +goose StatementEnd
