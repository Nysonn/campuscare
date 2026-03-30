ALTER TABLE campaign_attachments
    ADD COLUMN IF NOT EXISTS label TEXT NOT NULL DEFAULT 'Document';
