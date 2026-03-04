CREATE TABLE crisis_flags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID,
    message TEXT,
    created_at TIMESTAMP DEFAULT now()
);