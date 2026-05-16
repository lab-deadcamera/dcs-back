-- +goose Up
ALTER TABLE users
    ADD COLUMN role_id   INT          NOT NULL DEFAULT 4 REFERENCES roles(id),
    ADD COLUMN user_name VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN email     VARCHAR(255) NOT NULL DEFAULT '';

-- Set existing users to USER role (id=4, level=3)
UPDATE users SET role_id = (SELECT id FROM roles WHERE level = 3);

CREATE INDEX idx_users_role_id ON users(role_id);

-- +goose Down
DROP INDEX IF EXISTS idx_users_role_id;
ALTER TABLE users
    DROP COLUMN IF EXISTS role_id,
    DROP COLUMN IF EXISTS user_name,
    DROP COLUMN IF EXISTS email;
