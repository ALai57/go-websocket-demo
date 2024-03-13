-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE live_connections (
    id UUID NOT NULL DEFAULT uuid_generate_v1(),
    name text,
    created_at timestamp,
    extension text,
    PRIMARY KEY(id)
);

-- +goose Down
DROP TABLE live_connections
