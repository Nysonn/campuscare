CREATE TABLE general_pool_donations (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    donor_name     TEXT        NOT NULL,
    donor_email    TEXT        NOT NULL,
    donor_phone    TEXT        NOT NULL DEFAULT '',
    amount         BIGINT      NOT NULL,
    message        TEXT        NOT NULL DEFAULT '',
    payment_method TEXT        NOT NULL,
    is_anonymous   BOOLEAN     NOT NULL DEFAULT false,
    status         TEXT        NOT NULL DEFAULT 'success',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_general_pool_created ON general_pool_donations(created_at DESC);
