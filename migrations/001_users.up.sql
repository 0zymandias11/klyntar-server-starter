-- Users table: one row per account.
-- key_fingerprint is the primary SSH key used at registration.
-- user_keys holds all keys (including this one) for multi-device support.
CREATE TABLE users (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email            TEXT        NOT NULL UNIQUE,
    username         TEXT        NOT NULL UNIQUE,
    key_fingerprint  TEXT        NOT NULL UNIQUE,
    device_mac       TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- user_keys: additional SSH keys for the same account (new devices).
-- The fingerprint from registration is also inserted here so every
-- active key can be looked up from a single table.
CREATE TABLE user_keys (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_fingerprint  TEXT        NOT NULL UNIQUE,
    device_name      TEXT        NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_keys_user_id ON user_keys(user_id);
