-- Sponsor request status enum
CREATE TYPE sponsor_request_status AS ENUM ('pending', 'accepted', 'declined');

-- Students who choose to become sponsors
CREATE TABLE sponsor_profiles (
  user_id      UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  what_i_offer TEXT NOT NULL DEFAULT '',
  is_active    BOOLEAN NOT NULL DEFAULT TRUE,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Requests from one student to a sponsor (1-to-1)
CREATE TABLE sponsor_requests (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  requester_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  sponsor_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  status       sponsor_request_status NOT NULL DEFAULT 'pending',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(requester_id, sponsor_id)
);

-- Active sponsorships (one sponsor : one sponsee at a time)
CREATE TABLE sponsorships (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sponsor_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  sponsee_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  stream_channel_id TEXT NOT NULL,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  terminated_at     TIMESTAMPTZ,
  UNIQUE(sponsor_id, sponsee_id)
);