-- name: EnqueueRun :one
INSERT INTO runs (
    task_id,
    scheduled_at,
    status,
    source,
    created_at
)
VALUES ($1, $2, $3, $4, NOW())
RETURNING *;

-- name: ListPendingRuns :many
SELECT * FROM runs
WHERE status = 'QUEUED'
ORDER BY scheduled_at ASC;

-- name: GetLastScheduledForTask :one
SELECT scheduled_at
FROM runs
WHERE task_id = $1
ORDER BY scheduled_at DESC
LIMIT 1;

-- name: GetDueRuns :many
SELECT * FROM runs
WHERE status = 'QUEUED'
  AND scheduled_at <= NOW()
ORDER BY scheduled_at ASC;

-- name: MarkRunCompleted :exec
UPDATE runs
SET status = 'COMPLETED',
    finished_at = NOW()
WHERE id = $1;

-- name: MarkAsRunning :exec
UPDATE runs
SET status = 'RUNNING'
WHERE id = $1 AND status = "QUEUED";

-- name: MarkRunFailed :exec
UPDATE runs
SET status = 'FAILED',
    finished_at = NOW()
WHERE id = $1;
