-- name: AddJob :one
INSERT INTO jobs (id, entry_id, name, status, cron_spec, scheduled_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetJob :one
SELECT * FROM jobs WHERE id = $1;

-- name: ListJobs :many
SELECT *
FROM jobs
ORDER BY scheduled_at ASC
LIMIT $1 OFFSET $2;

-- name: UpdateJob :one
UPDATE jobs
SET
  entry_id     = COALESCE(sqlc.narg('entry_id'), entry_id),
  name         = COALESCE(sqlc.narg('name'),     name),
  status       = COALESCE(sqlc.narg('status'),   status),
  cron_spec    = COALESCE(sqlc.narg('cron_spec'),cron_spec),
  scheduled_at = COALESCE(sqlc.narg('scheduled_at'), scheduled_at)
WHERE id = $1
RETURNING *;

-- name: UpdateJobStatus :one
UPDATE jobs
SET status = $2
WHERE id = $1
RETURNING *;

-- name: UpdateJobEntryAndNext :one
UPDATE jobs
SET entry_id = $2, scheduled_at = $3
WHERE id = $1
RETURNING *;

-- name: DeleteJob :exec
DELETE FROM jobs WHERE id = $1;