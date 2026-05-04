ALTER TABLE reports
    ADD COLUMN IF NOT EXISTS wants_followup BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS followup_email TEXT;

CREATE TABLE IF NOT EXISTS pool_assignments (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id   UUID        NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    sponsor_id  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at    TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS welfare_reports (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id       UUID        NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    submitted_by    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    week_of         DATE        NOT NULL,
    wellbeing_score INT         NOT NULL CHECK (wellbeing_score BETWEEN 1 AND 5),
    observations    TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(report_id, submitted_by, week_of)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pool_assignments_active_report
    ON pool_assignments(report_id)
    WHERE ended_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_pool_assignments_active_sponsor
    ON pool_assignments(sponsor_id)
    WHERE ended_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_reports_followup_email
    ON reports (LOWER(followup_email))
    WHERE wants_followup = true;

CREATE INDEX IF NOT EXISTS idx_welfare_reports_report_created
    ON welfare_reports(report_id, created_at DESC);