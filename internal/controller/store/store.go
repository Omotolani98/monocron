package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// DB is the controller database abstraction.
type DB struct {
	db *sql.DB
}

// New opens a PostgreSQL connection and applies migrations.
func New(databaseURL string) (*DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	s := &DB{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return s, nil
}

// Close closes the database connection.
func (s *DB) Close() error { return s.db.Close() }

//go:embed migrations/controller/001_initial.sql
var migrationSQL string

func (s *DB) migrate() error {
	// Synchronous migration for development. In production use golang-migrate or similar.
	_, err := s.db.Exec(migrationSQL)
	return err
}

// CreateSchedule inserts a new schedule.
func (s *DB) CreateSchedule(ctx context.Context, req contracts.CreateScheduleRequest) (contracts.ScheduleSpec, error) {
	id := uuid.New()
	now := time.Now().UTC()
	labels, _ := json.Marshal(req.Labels)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO schedules(id, name, cron, timezone, timeout, command, labels, description, enabled, created_at, updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		id, req.Name, req.Cron, req.Timezone, req.Timeout, pq.Array(req.Command), labels, req.Description, true, now, now,
	)
	if err != nil {
		return contracts.ScheduleSpec{}, err
	}
	return s.GetSchedule(ctx, id)
}

// GetSchedule returns a schedule by ID.
func (s *DB) GetSchedule(ctx context.Context, id uuid.UUID) (contracts.ScheduleSpec, error) {
	var spec contracts.ScheduleSpec
	var labels []byte
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, cron, timezone, timeout, command, labels, description, enabled, created_at, updated_at
		 FROM schedules WHERE id=$1 AND deleted_at IS NULL`, id)
	err := row.Scan(&spec.ID, &spec.Name, &spec.Cron, &spec.Timezone, &spec.Timeout, pq.Array(&spec.Command), &labels, &spec.Description, &spec.Enabled, &spec.CreatedAt, &spec.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return spec, fmt.Errorf("schedule not found")
		}
		return spec, err
	}
	_ = json.Unmarshal(labels, &spec.Labels)
	return spec, nil
}

// UpdateSchedule updates a schedule.
func (s *DB) UpdateSchedule(ctx context.Context, id uuid.UUID, req contracts.UpdateScheduleRequest) (contracts.ScheduleSpec, error) {
	updates := map[string]any{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Cron != "" {
		updates["cron"] = req.Cron
	}
	if req.Timezone != "" {
		updates["timezone"] = req.Timezone
	}
	if req.Timeout != "" {
		updates["timeout"] = req.Timeout
	}
	if req.Command != nil {
		updates["command"] = pq.Array(req.Command)
	}
	if req.Labels != nil {
		labels, _ := json.Marshal(req.Labels)
		updates["labels"] = labels
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if len(updates) == 0 {
		return s.GetSchedule(ctx, id)
	}
	updates["updated_at"] = time.Now().UTC()

	// Build dynamic query
	query := "UPDATE schedules SET "
	args := []any{}
	i := 1
	for col, val := range updates {
		if i > 1 {
			query += ", "
		}
		query += fmt.Sprintf("%s=$%d", col, i)
		args = append(args, val)
		i++
	}
	query += fmt.Sprintf(" WHERE id=$%d AND deleted_at IS NULL", i)
	args = append(args, id)

	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return contracts.ScheduleSpec{}, err
	}
	return s.GetSchedule(ctx, id)
}

// DeleteSchedule soft-deletes a schedule.
func (s *DB) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE schedules SET deleted_at=$1, updated_at=$1 WHERE id=$2 AND deleted_at IS NULL`,
		time.Now().UTC(), id)
	return err
}

// ListSchedules returns schedules with optional cursor pagination.
func (s *DB) ListSchedules(ctx context.Context, cursor string, limit int) ([]contracts.ScheduleSpec, string, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	args := []any{limit}
	where := "WHERE deleted_at IS NULL"
	if cursor != "" {
		where += " AND created_at < $2"
		args = append(args, cursor)
	}
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT id, name, cron, timezone, timeout, command, labels, description, enabled, created_at, updated_at
		 FROM schedules %s ORDER BY created_at DESC LIMIT $1`, where), args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var items []contracts.ScheduleSpec
	var last time.Time
	for rows.Next() {
		var spec contracts.ScheduleSpec
		var labels []byte
		if err := rows.Scan(&spec.ID, &spec.Name, &spec.Cron, &spec.Timezone, &spec.Timeout, pq.Array(&spec.Command), &labels, &spec.Description, &spec.Enabled, &spec.CreatedAt, &spec.UpdatedAt); err != nil {
			return nil, "", err
		}
		_ = json.Unmarshal(labels, &spec.Labels)
		items = append(items, spec)
		last = spec.CreatedAt
	}
	var next string
	if len(items) == limit && !last.IsZero() {
		next = last.Format(time.RFC3339Nano)
	}
	return items, next, rows.Err()
}

// CreateEnrollmentToken creates a new enrollment token and returns its plaintext value.
func (s *DB) CreateEnrollmentToken(ctx context.Context, req contracts.CreateEnrollmentTokenRequest) (string, contracts.CreateEnrollmentTokenResponse, error) {
	plaintext, err := generateToken()
	if err != nil {
		return "", contracts.CreateEnrollmentTokenResponse{}, err
	}
	hash := hashToken(plaintext)
	id := uuid.New()
	expires := time.Now().UTC().Add(24 * time.Hour)
	if req.Expires != nil {
		expires = *req.Expires
	}
	labels, _ := json.Marshal(req.Labels)
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO enrollment_tokens(id, token_hash, scope, labels, expires_at, created_at)
		VALUES($1,$2,$3,$4,$5,$6)`,
		id, hash, req.Scope, labels, expires, time.Now().UTC())
	if err != nil {
		return "", contracts.CreateEnrollmentTokenResponse{}, err
	}
	return plaintext, contracts.CreateEnrollmentTokenResponse{Token: plaintext, ExpiresAt: expires}, nil
}

// ConsumeEnrollmentToken validates a plaintext token and marks it consumed, returning its metadata.
func (s *DB) ConsumeEnrollmentToken(ctx context.Context, plaintext string) (labels contracts.Labels, err error) {
	hash := hashToken(plaintext)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var id uuid.UUID
	var tokenLabels []byte
	var expires time.Time
	var consumed sql.NullTime
	err = tx.QueryRowContext(ctx,
		`SELECT id, labels, expires_at, consumed_at FROM enrollment_tokens WHERE token_hash=$1 FOR UPDATE`,
		hash).Scan(&id, &tokenLabels, &expires, &consumed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("invalid or expired token")
		}
		return nil, err
	}
	if consumed.Valid {
		return nil, fmt.Errorf("token already consumed")
	}
	if time.Now().UTC().After(expires) {
		return nil, fmt.Errorf("token expired")
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE enrollment_tokens SET consumed_at=$1, consumed_by=$2 WHERE id=$3`,
		time.Now().UTC(), uuid.Nil, id); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(tokenLabels, &labels)
	return labels, tx.Commit()
}

// CreateRunner creates a new runner with an access token.
func (s *DB) CreateRunner(ctx context.Context, runnerType contracts.RunnerType, labels contracts.Labels) (contracts.Runner, string, error) {
	plaintext, err := generateToken()
	if err != nil {
		return contracts.Runner{}, "", err
	}
	id := uuid.New()
	now := time.Now().UTC()
	labelsJSON, _ := json.Marshal(labels)
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO runners(id, name, runner_type, labels, state, access_token_hash, created_at, updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		id, "", runnerType, labelsJSON, contracts.RunnerStatePending, hashToken(plaintext), now, now)
	if err != nil {
		return contracts.Runner{}, "", err
	}
	runner, err := s.GetRunner(ctx, id)
	return runner, plaintext, err
}

// EnrollRunner consumes an enrollment token and creates a runner in one transaction.
// If runner creation fails, the token consumption is rolled back.
func (s *DB) EnrollRunner(ctx context.Context, plaintextToken string) (contracts.Runner, string, error) {
	hash := hashToken(plaintextToken)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contracts.Runner{}, "", err
	}
	defer tx.Rollback()

	var tokenID uuid.UUID
	var tokenLabels []byte
	var expires time.Time
	var consumed sql.NullTime
	err = tx.QueryRowContext(ctx,
		`SELECT id, labels, expires_at, consumed_at FROM enrollment_tokens WHERE token_hash=$1 FOR UPDATE`,
		hash).Scan(&tokenID, &tokenLabels, &expires, &consumed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.Runner{}, "", fmt.Errorf("invalid or expired token")
		}
		return contracts.Runner{}, "", err
	}
	if consumed.Valid {
		return contracts.Runner{}, "", fmt.Errorf("token already consumed")
	}
	if time.Now().UTC().After(expires) {
		return contracts.Runner{}, "", fmt.Errorf("token expired")
	}

	var labels contracts.Labels
	_ = json.Unmarshal(tokenLabels, &labels)
	if labels == nil {
		labels = contracts.Labels{}
	}

	runnerType := contracts.RunnerTypeBareMetal
	if t := labels["type"]; t != "" {
		runnerType = contracts.RunnerType(t)
		delete(labels, "type")
	}

	accessTokenPlaintext, err := generateToken()
	if err != nil {
		return contracts.Runner{}, "", err
	}
	runnerID := uuid.New()
	now := time.Now().UTC()
	labelsJSON, _ := json.Marshal(labels)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO runners(id, name, runner_type, labels, state, access_token_hash, created_at, updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		runnerID, "", runnerType, labelsJSON, contracts.RunnerStatePending, hashToken(accessTokenPlaintext), now, now)
	if err != nil {
		return contracts.Runner{}, "", err
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE enrollment_tokens SET consumed_at=$1, consumed_by=$2 WHERE id=$3`,
		now, runnerID, tokenID)
	if err != nil {
		return contracts.Runner{}, "", err
	}

	if err := tx.Commit(); err != nil {
		return contracts.Runner{}, "", err
	}

	return contracts.Runner{
		ID:        runnerID,
		Type:      runnerType,
		Labels:    labels,
		State:     contracts.RunnerStatePending,
		CreatedAt: now,
	}, accessTokenPlaintext, nil
}

// GetRunner returns a runner by ID.
func (s *DB) GetRunner(ctx context.Context, id uuid.UUID) (contracts.Runner, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, runner_type, labels, state, version, daemon_version, last_heartbeat, created_at
		 FROM runners WHERE id=$1`, id)
	r, err := scanRunner(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.Runner{}, fmt.Errorf("runner not found")
		}
		return contracts.Runner{}, err
	}
	return r, nil
}

// ListRunners returns runners.
func (s *DB) ListRunners(ctx context.Context, cursor string, limit int) ([]contracts.Runner, string, error) {
	if limit <= 0 {
		limit = 50
	}
	args := []any{limit}
	where := ""
	if cursor != "" {
		where = " WHERE created_at < $2"
		args = append(args, cursor)
	}
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT id, name, runner_type, labels, state, version, daemon_version, last_heartbeat, created_at
		 FROM runners%s ORDER BY created_at DESC LIMIT $1`, where), args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var items []contracts.Runner
	var last time.Time
	for rows.Next() {
		r, err := scanRunner(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, r)
		last = r.CreatedAt
	}
	var next string
	if len(items) == limit && !last.IsZero() {
		next = last.Format(time.RFC3339Nano)
	}
	return items, next, rows.Err()
}

type runnerScanner interface {
	Scan(dest ...any) error
}

// scanRunner scans a runner row, handling nullable version/daemon_version/last_heartbeat columns.
func scanRunner(s runnerScanner) (contracts.Runner, error) {
	var r contracts.Runner
	var labels []byte
	var version, daemonVersion sql.NullString
	var lastHeartbeat sql.NullTime
	err := s.Scan(&r.ID, &r.Name, &r.Type, &labels, &r.State, &version, &daemonVersion, &lastHeartbeat, &r.CreatedAt)
	if err != nil {
		return r, err
	}
	r.Version = version.String
	r.DaemonVersion = daemonVersion.String
	if lastHeartbeat.Valid {
		r.LastHeartbeat = lastHeartbeat.Time
	}
	_ = json.Unmarshal(labels, &r.Labels)
	return r, nil
}

// UpdateRunnerState updates runner state.
func (s *DB) UpdateRunnerState(ctx context.Context, id uuid.UUID, state contracts.RunnerState) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE runners SET state=$1, updated_at=$2 WHERE id=$3`, state, time.Now().UTC(), id)
	return err
}

// UpdateRunnerHeartbeat updates heartbeat and version metadata.
func (s *DB) UpdateRunnerHeartbeat(ctx context.Context, id uuid.UUID, req contracts.HeartbeatRequest) error {
	labelsJSON, _ := json.Marshal(req.Labels)
	_, err := s.db.ExecContext(ctx,
		`UPDATE runners SET labels=$1, version=$2, daemon_version=$3, last_heartbeat=$4, updated_at=$4 WHERE id=$5`,
		labelsJSON, req.Version, req.DaemonVersion, time.Now().UTC(), id)
	return err
}

// ValidateAccessToken checks an access token and returns the runner ID.
func (s *DB) ValidateAccessToken(ctx context.Context, plaintext string) (uuid.UUID, error) {
	hash := hashToken(plaintext)
	var id uuid.UUID
	var state contracts.RunnerState
	err := s.db.QueryRowContext(ctx,
		`SELECT id, state FROM runners WHERE access_token_hash=$1`, hash).Scan(&id, &state)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("invalid token")
		}
		return uuid.Nil, err
	}
	if state == contracts.RunnerStateRevoked {
		return uuid.Nil, fmt.Errorf("runner revoked")
	}
	return id, nil
}

// DeleteRunner removes a runner.
func (s *DB) DeleteRunner(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM runners WHERE id=$1`, id)
	return err
}

// CreateAssignment inserts a new assignment.
func (s *DB) CreateAssignment(ctx context.Context, scheduleID, runnerID uuid.UUID, generation int64) (contracts.Assignment, error) {
	id := uuid.New()
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO assignments(id, schedule_id, runner_id, state, generation, created_at, updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT(schedule_id, runner_id) DO UPDATE SET
			state=EXCLUDED.state,
			generation=EXCLUDED.generation,
			updated_at=EXCLUDED.updated_at`,
		id, scheduleID, runnerID, contracts.AssignmentStatePending, generation, now, now)
	if err != nil {
		return contracts.Assignment{}, err
	}
	return s.GetAssignment(ctx, id)
}

// GetAssignment returns an assignment by ID.
func (s *DB) GetAssignment(ctx context.Context, id uuid.UUID) (contracts.Assignment, error) {
	var a contracts.Assignment
	err := s.db.QueryRowContext(ctx,
		`SELECT id, schedule_id, runner_id, state, generation, created_at, updated_at FROM assignments WHERE id=$1`, id).Scan(
		&a.ID, &a.ScheduleID, &a.RunnerID, &a.State, &a.Generation, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return a, fmt.Errorf("assignment not found")
		}
		return a, err
	}
	return a, nil
}

// ListAssignmentsForRunner returns assignments for a runner with generation greater than after.
func (s *DB) ListAssignmentsForRunner(ctx context.Context, runnerID uuid.UUID, after int64) ([]contracts.Assignment, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, schedule_id, runner_id, state, generation, created_at, updated_at
		 FROM assignments WHERE runner_id=$1 AND generation>$2 ORDER BY generation ASC`,
		runnerID, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []contracts.Assignment
	for rows.Next() {
		var a contracts.Assignment
		if err := rows.Scan(&a.ID, &a.ScheduleID, &a.RunnerID, &a.State, &a.Generation, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpdateAssignmentState updates assignment state.
func (s *DB) UpdateAssignmentState(ctx context.Context, id uuid.UUID, state contracts.AssignmentState) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE assignments SET state=$1, updated_at=$2 WHERE id=$3`, state, time.Now().UTC(), id)
	return err
}

// CreateExecution inserts a new execution.
func (s *DB) CreateExecution(ctx context.Context, assignmentID, scheduleID, runnerID uuid.UUID) (contracts.Execution, error) {
	id := uuid.New()
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO executions(id, assignment_id, schedule_id, runner_id, state, created_at, updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7)`,
		id, assignmentID, scheduleID, runnerID, contracts.ExecutionStateQueued, now, now)
	if err != nil {
		return contracts.Execution{}, err
	}
	return s.GetExecution(ctx, id)
}

// GetExecution returns an execution by ID.
func (s *DB) GetExecution(ctx context.Context, id uuid.UUID) (contracts.Execution, error) {
	var e contracts.Execution
	var exit sql.NullInt32
	err := s.db.QueryRowContext(ctx,
		`SELECT id, assignment_id, schedule_id, runner_id, state, started_at, finished_at, exit_code, error_message, created_at
		 FROM executions WHERE id=$1`, id).Scan(
		&e.ID, &e.AssignmentID, &e.ScheduleID, &e.RunnerID, &e.State, &e.StartedAt, &e.FinishedAt, &exit, &e.ErrorMessage, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return e, fmt.Errorf("execution not found")
		}
		return e, err
	}
	if exit.Valid {
		code := int(exit.Int32)
		e.ExitCode = &code
	}
	return e, nil
}

// ListExecutions returns executions.
func (s *DB) ListExecutions(ctx context.Context, cursor string, limit int) ([]contracts.Execution, string, error) {
	if limit <= 0 {
		limit = 50
	}
	args := []any{limit}
	where := ""
	if cursor != "" {
		where = " WHERE created_at < $2"
		args = append(args, cursor)
	}
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT id, assignment_id, schedule_id, runner_id, state, started_at, finished_at, exit_code, error_message, created_at
		 FROM executions%s ORDER BY created_at DESC LIMIT $1`, where), args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var items []contracts.Execution
	var last time.Time
	for rows.Next() {
		var e contracts.Execution
		var exit sql.NullInt32
		if err := rows.Scan(&e.ID, &e.AssignmentID, &e.ScheduleID, &e.RunnerID, &e.State, &e.StartedAt, &e.FinishedAt, &exit, &e.ErrorMessage, &e.CreatedAt); err != nil {
			return nil, "", err
		}
		if exit.Valid {
			code := int(exit.Int32)
			e.ExitCode = &code
		}
		items = append(items, e)
		last = e.CreatedAt
	}
	var next string
	if len(items) == limit && !last.IsZero() {
		next = last.Format(time.RFC3339Nano)
	}
	return items, next, rows.Err()
}

// UpdateExecutionState updates execution state.
func (s *DB) UpdateExecutionState(ctx context.Context, id uuid.UUID, state contracts.ExecutionState, started, finished *time.Time, exitCode *int, errorMessage string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE executions SET state=$1, started_at=$2, finished_at=$3, exit_code=$4, error_message=$5, updated_at=$6 WHERE id=$7`,
		state, started, finished, nullableInt32(exitCode), errorMessage, time.Now().UTC(), id)
	return err
}

// SaveLogChunks persists log chunks idempotently.
func (s *DB) SaveLogChunks(ctx context.Context, chunks []contracts.LogChunk) error {
	for _, c := range chunks {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO log_chunks(id, execution_id, stream, sequence, payload, emitted_at)
			VALUES($1,$2,$3,$4,$5,$6)
			ON CONFLICT(execution_id, stream, sequence) DO UPDATE SET payload=EXCLUDED.payload`,
			c.ID, c.ExecutionID, c.Stream, c.Sequence, c.Payload, c.EmittedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetExecutionLogChunks returns log chunks for an execution ordered by sequence.
func (s *DB) GetExecutionLogChunks(ctx context.Context, execID uuid.UUID) ([]contracts.LogChunk, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, execution_id, stream, sequence, payload, emitted_at
		 FROM log_chunks WHERE execution_id=$1 ORDER BY sequence ASC`, execID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []contracts.LogChunk
	for rows.Next() {
		var c contracts.LogChunk
		var idStr, eid string
		if err := rows.Scan(&idStr, &eid, &c.Stream, &c.Sequence, &c.Payload, &c.EmittedAt); err != nil {
			return nil, err
		}
		c.ID, _ = uuid.Parse(idStr)
		c.ExecutionID, _ = uuid.Parse(eid)
		out = append(out, c)
	}
	return out, rows.Err()
}

// Audit records an audit event.
func (s *DB) Audit(ctx context.Context, actor, action, target, reason string, details any) error {
	detailsJSON, _ := json.Marshal(details)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_events(id, actor, action, target, reason, details, created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7)`,
		uuid.New(), actor, action, target, reason, detailsJSON, time.Now().UTC())
	return err
}

// ListAuditEvents returns audit events.
func (s *DB) ListAuditEvents(ctx context.Context, cursor string, limit int) ([]contracts.AuditEvent, string, error) {
	if limit <= 0 {
		limit = 50
	}
	args := []any{limit}
	where := ""
	if cursor != "" {
		where = " WHERE created_at < $2"
		args = append(args, cursor)
	}
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT id, actor, action, target, reason, details, created_at
		 FROM audit_events%s ORDER BY created_at DESC LIMIT $1`, where), args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var items []contracts.AuditEvent
	var last time.Time
	for rows.Next() {
		var e contracts.AuditEvent
		if err := rows.Scan(&e.ID, &e.Actor, &e.Action, &e.Target, &e.Reason, &e.Details, &e.CreatedAt); err != nil {
			return nil, "", err
		}
		items = append(items, e)
		last = e.CreatedAt
	}
	var next string
	if len(items) == limit && !last.IsZero() {
		next = last.Format(time.RFC3339Nano)
	}
	return items, next, rows.Err()
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "mt_" + hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func nullableInt32(n *int) sql.NullInt32 {
	if n == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(*n), Valid: true}
}
