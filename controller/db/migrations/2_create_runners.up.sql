CREATE TABLE runners (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kind TEXT NOT NULL,                 -- vm | docker | bare-metal
  status TEXT NOT NULL DEFAULT 'live',-- live | dead
  last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_runners_last_seen ON runners(last_seen DESC);

