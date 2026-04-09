CREATE TABLE voids(
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR         NOT NULL UNIQUE,
    creator_id          UUID            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    description         VARCHAR         NOT NULL UNIQUE,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
)