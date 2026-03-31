ALTER TABLE campaigns
    DROP COLUMN IF EXISTS urgency_level,
    DROP COLUMN IF EXISTS beneficiary_type,
    DROP COLUMN IF EXISTS beneficiary_name,
    DROP COLUMN IF EXISTS verification_contact_name,
    DROP COLUMN IF EXISTS verification_contact_info,
    DROP COLUMN IF EXISTS beneficiary_org_name,
    DROP COLUMN IF EXISTS bank_name,
    DROP COLUMN IF EXISTS account_number,
    DROP COLUMN IF EXISTS account_holder_name,
    DROP COLUMN IF EXISTS account_status;
