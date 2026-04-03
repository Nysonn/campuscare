ALTER TABLE counselor_profiles
    DROP COLUMN IF EXISTS location,
    DROP COLUMN IF EXISTS age,
    DROP COLUMN IF EXISTS years_of_experience,
    DROP COLUMN IF EXISTS licence_url,
    DROP COLUMN IF EXISTS verification_status;
