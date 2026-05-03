CREATE TABLE blogs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT        NOT NULL,
    description TEXT        NOT NULL,
    content     TEXT        NOT NULL,
    image_url   TEXT        NOT NULL DEFAULT '',
    author      TEXT        NOT NULL DEFAULT 'CampusCare Team',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
