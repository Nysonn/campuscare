CREATE TABLE chatbot_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    role TEXT CHECK (role IN ('user','assistant')),
    content TEXT,
    created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE chatbot_usage (
    user_id UUID PRIMARY KEY,
    count INT,
    last_reset TIMESTAMP
);
