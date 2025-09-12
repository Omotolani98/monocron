-- name: UpsertTask :one
INSERT INTO tasks (
    name,
    schedule,
    timezone,
    concurrency_policy,
    retries,
    backoff,
    catchup,
    catchup_window,
    executor,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
ON CONFLICT (name) DO UPDATE
SET
    schedule           = EXCLUDED.schedule,
    timezone           = EXCLUDED.timezone,
    concurrency_policy = EXCLUDED.concurrency_policy,
    retries            = EXCLUDED.retries,
    backoff            = EXCLUDED.backoff,
    catchup            = EXCLUDED.catchup,
    catchup_window     = EXCLUDED.catchup_window,
    executor           = EXCLUDED.executor,
    updated_at         = NOW()
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks
WHERE name = $1
LIMIT 1;

-- name: GetTaskById :one
SELECT * FROM tasks
WHERE id = $1
LIMIT 1;

-- name: ListTasks :many
SELECT * FROM tasks
ORDER BY created_at DESC;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE name = $1;