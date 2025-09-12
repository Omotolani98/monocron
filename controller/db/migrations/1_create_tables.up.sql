CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  schedule TEXT NOT NULL,
  timezone TEXT NOT NULL DEFAULT 'UTC',
  concurrency_policy TEXT NOT NULL DEFAULT 'Allow',
  retries INT DEFAULT 0,
  backoff TEXT,
  catchup BOOLEAN DEFAULT false,
  catchup_window INTERVAL,
  executor JSONB NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  scheduled_at TIMESTAMPTZ NOT NULL,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'pending',   -- pending|running|success|failed
  source TEXT NOT NULL DEFAULT 'regular',   -- regular|catchup
  logs JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (task_id, scheduled_at)
);

CREATE TABLE logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
  log_line TEXT NOT NULL,
  logged_at TIMESTAMPTZ DEFAULT NOW()
);