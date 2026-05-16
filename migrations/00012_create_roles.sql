-- +goose Up
CREATE TABLE roles (
    id    SERIAL PRIMARY KEY,
    name  VARCHAR(50) UNIQUE NOT NULL,
    level INT UNIQUE NOT NULL
);

INSERT INTO roles (name, level) VALUES
    ('SUPER_ADMIN', 0),
    ('ADMIN', 1),
    ('SUPERVISOR', 2),
    ('USER', 3);

-- +goose Down
DROP TABLE IF EXISTS roles CASCADE;
