DROP INDEX IF EXISTS idx_welfare_reports_report_created;
DROP INDEX IF EXISTS idx_reports_followup_email;
DROP INDEX IF EXISTS idx_pool_assignments_active_sponsor;
DROP INDEX IF EXISTS idx_pool_assignments_active_report;

DROP TABLE IF EXISTS welfare_reports;
DROP TABLE IF EXISTS pool_assignments;

ALTER TABLE reports
    DROP COLUMN IF EXISTS followup_email,
    DROP COLUMN IF EXISTS wants_followup;