CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    cron TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    timeout TEXT NOT NULL DEFAULT '30s',
    command TEXT[] NOT NULL,
    labels JSONB NOT NULL DEFAULT '{}',
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS enrollment_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token_hash TEXT NOT NULL UNIQUE,
    scope TEXT,
    labels JSONB NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    consumed_at TIMESTAMPTZ,
    consumed_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS runners (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT,
    runner_type TEXT NOT NULL DEFAULT 'bare_metal',
    labels JSONB NOT NULL DEFAULT '{}',
    state TEXT NOT NULL DEFAULT 'pending',
    version TEXT,
    daemon_version TEXT,
    last_heartbeat TIMESTAMPTZ,
    access_token_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    schedule_id UUID NOT NULL REFERENCES schedules(id),
    runner_id UUID NOT NULL REFERENCES runners(id),
    state TEXT NOT NULL DEFAULT 'pending',
    generation BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(schedule_id, runner_id)
);

CREATE INDEX IF NOT EXISTS idx_assignments_runner ON assignments(runner_id);
CREATE INDEX IF NOT EXISTS idx_assignments_generation ON assignments(generation);

CREATE TABLE IF NOT EXISTS executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    assignment_id UUID NOT NULL REFERENCES assignments(id),
    schedule_id UUID NOT NULL,
    runner_id UUID NOT NULL,
    state TEXT NOT NULL DEFAULT 'queued',
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    exit_code INTEGER,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_executions_assignment ON executions(assignment_id);
CREATE INDEX IF NOT EXISTS idx_executions_runner ON executions(runner_id);

CREATE TABLE IF NOT EXISTS log_chunks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    execution_id UUID NOT NULL REFERENCES executions(id),
    stream TEXT NOT NULL,
    sequence BIGINT NOT NULL,
    payload BYTEA NOT NULL,
    emitted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(execution_id, stream, sequence)
);

CREATE INDEX IF NOT EXISTS idx_log_chunks_execution ON log_chunks(execution_id);

CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    target TEXT NOT NULL,
    reason TEXT,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_events_target ON audit_events(target);
CREATE INDEX IF NOT EXISTS idx_audit_events_created_at ON audit_events(created_at);
