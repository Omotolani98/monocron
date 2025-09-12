-- name: InsertLog :one
INSERT INTO logs (
    run_id,
    log_line,
    logged_at
)
VALUES ($1, $2, NOW())
RETURNING *;

-- name: ListLogsByRun :many
SELECT * FROM logs
WHERE run_id = $1
ORDER BY logged_at ASC;
