ALTER TABLE behaviour_goals
  DROP COLUMN IF EXISTS last_motivation_sent,
  DROP COLUMN IF EXISTS missed_notified_date;
