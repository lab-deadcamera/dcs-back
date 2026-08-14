-- +goose Up
-- +goose StatementBegin

-- ─── Push Subscriptions (Web Push) ───────────────────────────────
-- One row per user device subscribed to receive push notifications.
-- user_id matches users.id (SERIAL). The endpoint is the browser's
-- push service URL; p256dh/auth are the subscription encryption keys.
CREATE TABLE push_subscriptions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint    TEXT NOT NULL,
    p256dh      TEXT NOT NULL,
    auth        TEXT NOT NULL,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, endpoint)
);

CREATE INDEX idx_push_subscriptions_user ON push_subscriptions(user_id);

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS push_subscriptions;
