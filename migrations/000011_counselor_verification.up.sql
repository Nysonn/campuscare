ALTER TABLE counselor_profiles
    ADD COLUMN IF NOT EXISTS location             TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS age                  INT,
    ADD COLUMN IF NOT EXISTS years_of_experience  TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS licence_url          TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS verification_status  TEXT    NOT NULL DEFAULT 'pending';

-- Grandfather existing counsellors as approved so they aren't locked out.
UPDATE counselor_profiles SET verification_status = 'approved' WHERE verification_status = 'pending';
