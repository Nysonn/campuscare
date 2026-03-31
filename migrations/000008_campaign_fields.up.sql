ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS urgency_level             TEXT NOT NULL DEFAULT 'normal',
    ADD COLUMN IF NOT EXISTS beneficiary_type          TEXT NOT NULL DEFAULT 'self',
    ADD COLUMN IF NOT EXISTS beneficiary_name          TEXT,
    ADD COLUMN IF NOT EXISTS verification_contact_name TEXT,
    ADD COLUMN IF NOT EXISTS verification_contact_info TEXT,
    ADD COLUMN IF NOT EXISTS beneficiary_org_name      TEXT,
    ADD COLUMN IF NOT EXISTS bank_name                 TEXT,
    ADD COLUMN IF NOT EXISTS account_number            TEXT,
    ADD COLUMN IF NOT EXISTS account_holder_name       TEXT,
    ADD COLUMN IF NOT EXISTS account_status            TEXT NOT NULL DEFAULT 'unverified';
