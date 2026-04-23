CREATE TABLE pool_disbursements (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  campaign_id UUID        NOT NULL REFERENCES campaigns(id),
  amount      BIGINT      NOT NULL CHECK (amount > 0),
  note        TEXT        NOT NULL DEFAULT '',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE pool_withdrawals (
  id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  amount           BIGINT      NOT NULL CHECK (amount > 0),
  destination_type TEXT        NOT NULL, -- 'bank' | 'mtn_momo' | 'airtel_money'
  destination_name TEXT        NOT NULL,
  account_number   TEXT        NOT NULL,
  note             TEXT        NOT NULL DEFAULT '',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
