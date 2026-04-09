CREATE TABLE posts(
    id              UUID        NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    title           VARCHAR     NOT NULL,
    content         VARCHAR     NOT NULL,  
    creator_id      UUID     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    void_id         UUID     NOT NULL REFERENCES voids(id) ON DELETE CASCADE,
    score           INT         NOT NULL DEFAULT 0
)