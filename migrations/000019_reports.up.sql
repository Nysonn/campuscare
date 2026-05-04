CREATE TABLE IF NOT EXISTS reports (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_name   TEXT,
    subject_name    TEXT        NOT NULL,
    subject_contact TEXT,
    university      TEXT,
    description     TEXT        NOT NULL,
    urgency         TEXT        NOT NULL DEFAULT 'medium'
                                CHECK (urgency IN ('low','medium','high','critical')),
    status          TEXT        NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending','reviewed','actioned','closed')),
    admin_notes     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
