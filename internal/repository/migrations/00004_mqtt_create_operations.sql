-- +goose Up
CREATE TABLE IF NOT EXISTS mqtt_create_operations (
    scope_key char(64) PRIMARY KEY,
    api_context text NOT NULL,
    principal text NOT NULL,
    operation_key text NOT NULL,
    fingerprint char(64) NOT NULL,
    resource_path text NOT NULL DEFAULT '',
    media_type text NOT NULL DEFAULT '',
    response bytea,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS mqtt_create_operations;
