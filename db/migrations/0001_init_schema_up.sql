-- +goose Up
-- Enable uuid type
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE jobs (
  id           uuid        PRIMARY KEY,            -- UUIDv7 generated in app
  entry_id     integer,                            -- robfig/cron EntryID (nullable until scheduled)
  name         text        NOT NULL,
  status       text        NOT NULL DEFAULT 'pending', -- plain text, default
  cron_spec    text        NOT NULL,
  scheduled_at timestamptz NOT NULL,               -- next run time computed from cron_spec (TZ-aware)
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);

-- Helpful indexes
CREATE INDEX IF NOT EXISTS jobs_status_idx       ON jobs (status);
CREATE INDEX IF NOT EXISTS jobs_scheduled_at_idx ON jobs (scheduled_at);

-- Prevent accidental duplicate registration of the same EntryID; allow NULLs to repeat
CREATE UNIQUE INDEX IF NOT EXISTS jobs_entry_id_uq
  ON jobs (entry_id) WHERE entry_id IS NOT NULL;

-- -- Keep updated_at fresh
-- CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
-- BEGIN
--   NEW.updated_at = now();
--   RETURN NEW;
-- END; $$ LANGUAGE plpgsql;

-- DROP TRIGGER IF EXISTS trg_jobs_updated_at ON jobs;
-- CREATE TRIGGER trg_jobs_updated_at
-- BEFORE UPDATE ON jobs
-- FOR EACH ROW EXECUTE PROCEDURE set_updated_at();