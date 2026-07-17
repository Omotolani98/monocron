package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/Omotolani98/monocron/internal/daemon/scheduler"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Store persists daemon assignments, executions, and log chunks.
type Store struct {
	db *sql.DB
}

// New opens the SQLite database and runs migrations.
func New(ctx context.Context, dsn string) (*Store, error) {
	if dsn == "" {
		dsn = ":memory:"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		return nil, fmt.Errorf("migrate sqlite: %w", err)
	}
	return s, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS assignments (
			id TEXT PRIMARY KEY,
			schedule_id TEXT NOT NULL,
			name TEXT NOT NULL,
			cron TEXT NOT NULL,
			timezone TEXT,
			timeout_ns INTEGER NOT NULL,
			command TEXT NOT NULL,
			generation INTEGER NOT NULL,
			enabled BOOLEAN NOT NULL,
			created_at TEXT NOT NULL
		);`,
			`CREATE TABLE IF NOT EXISTS executions (
			id TEXT PRIMARY KEY,
			assignment_id TEXT NOT NULL,
			schedule_id TEXT NOT NULL,
			state TEXT NOT NULL,
			started_at DATETIME,
			finished_at DATETIME,
			exit_code INTEGER,
			error_message TEXT,
			stdout BLOB,
			stderr BLOB,
			created_at DATETIME NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_executions_assignment ON executions(assignment_id);`,
		`CREATE TABLE IF NOT EXISTS log_chunks (
			id TEXT PRIMARY KEY,
			execution_id TEXT NOT NULL,
			stream TEXT NOT NULL,
			sequence INTEGER NOT NULL,
			payload BLOB NOT NULL,
			emitted_at TEXT NOT NULL,
			UNIQUE(execution_id, stream, sequence)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_log_chunks_execution ON log_chunks(execution_id);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// SaveAssignment persists or replaces an assignment.
func (s *Store) SaveAssignment(ctx context.Context, a scheduler.Assignment) error {
	cmd := joinCommand(a.Command)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO assignments(id, schedule_id, name, cron, timezone, timeout_ns, command, generation, enabled, created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			schedule_id=excluded.schedule_id,
			name=excluded.name,
			cron=excluded.cron,
			timezone=excluded.timezone,
			timeout_ns=excluded.timeout_ns,
			command=excluded.command,
			generation=excluded.generation,
			enabled=excluded.enabled,
			created_at=excluded.created_at`,
		a.ID.String(), a.ScheduleID.String(), a.Name, a.Cron, a.Timezone,
		a.Timeout.Nanoseconds(), cmd, a.Generation, a.Enabled, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// DeleteAssignment removes an assignment.
func (s *Store) DeleteAssignment(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM assignments WHERE id=?`, id.String())
	return err
}

// LoadAssignments returns all persisted assignments.
func (s *Store) LoadAssignments(ctx context.Context) ([]scheduler.Assignment, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, schedule_id, name, cron, timezone, timeout_ns, command, generation, enabled FROM assignments`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []scheduler.Assignment
	for rows.Next() {
		var a scheduler.Assignment
		var id, sid string
		var cmd string
		if err := rows.Scan(&id, &sid, &a.Name, &a.Cron, &a.Timezone, &a.Timeout, &cmd, &a.Generation, &a.Enabled); err != nil {
			return nil, err
		}
		a.ID, err = uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		a.ScheduleID, err = uuid.Parse(sid)
		if err != nil {
			return nil, err
		}
		a.Command = splitCommand(cmd)
		out = append(out, a)
	}
	return out, rows.Err()
}

// PersistExecution inserts or updates an execution record.
func (s *Store) PersistExecution(ctx context.Context, rec scheduler.ExecutionRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO executions(id, assignment_id, schedule_id, state, started_at, finished_at, exit_code, error_message, stdout, stderr, created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			state=excluded.state,
			started_at=excluded.started_at,
			finished_at=excluded.finished_at,
			exit_code=excluded.exit_code,
			error_message=excluded.error_message,
			stdout=excluded.stdout,
			stderr=excluded.stderr`,
		rec.ID.String(), rec.AssignmentID.String(), rec.ScheduleID.String(), string(rec.State),
		formatTime(rec.StartedAt), formatTime(rec.FinishedAt), nullableInt(rec.ExitCode),
		rec.ErrorMessage, rec.Stdout, rec.Stderr, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// AppendLogChunk persists a log chunk.
func (s *Store) AppendLogChunk(ctx context.Context, execID uuid.UUID, stream string, seq int64, payload []byte) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO log_chunks(id, execution_id, stream, sequence, payload, emitted_at)
		VALUES(?,?,?,?,?,?)
		ON CONFLICT(execution_id, stream, sequence) DO UPDATE SET payload=excluded.payload`,
		uuid.New().String(), execID.String(), stream, seq, payload, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// GetExecution returns an execution by ID.
func (s *Store) GetExecution(ctx context.Context, id uuid.UUID) (scheduler.ExecutionRecord, error) {
	var rec scheduler.ExecutionRecord
	var idStr, aid, sid string
	var exit sql.NullInt32
	row := s.db.QueryRowContext(ctx,
		`SELECT id, assignment_id, schedule_id, state, started_at, finished_at, exit_code, error_message, stdout, stderr
		 FROM executions WHERE id=?`, id.String())
	err := row.Scan(&idStr, &aid, &sid, &rec.State, &rec.StartedAt, &rec.FinishedAt, &exit, &rec.ErrorMessage, &rec.Stdout, &rec.Stderr)
	if err != nil {
		return rec, err
	}
	rec.ID, err = uuid.Parse(idStr)
	if err != nil {
		return rec, err
	}
	rec.AssignmentID, err = uuid.Parse(aid)
	if err != nil {
		return rec, err
	}
	rec.ScheduleID, err = uuid.Parse(sid)
	if err != nil {
		return rec, err
	}
	if exit.Valid {
		code := int(exit.Int32)
		rec.ExitCode = &code
	}
	return rec, nil
}

// ListExecutions returns executions for an assignment.
func (s *Store) ListExecutions(ctx context.Context, assignmentID uuid.UUID, limit int) ([]scheduler.ExecutionRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, assignment_id, schedule_id, state, started_at, finished_at, exit_code, error_message, stdout, stderr
		 FROM executions WHERE assignment_id=? ORDER BY created_at DESC LIMIT ?`,
		assignmentID.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []scheduler.ExecutionRecord
	for rows.Next() {
		var rec scheduler.ExecutionRecord
		var idStr, aid, sid string
		var exit sql.NullInt32
		if err := rows.Scan(&idStr, &aid, &sid, &rec.State, &rec.StartedAt, &rec.FinishedAt, &exit, &rec.ErrorMessage, &rec.Stdout, &rec.Stderr); err != nil {
			return nil, err
		}
		rec.ID, _ = uuid.Parse(idStr)
		rec.AssignmentID, _ = uuid.Parse(aid)
		rec.ScheduleID, _ = uuid.Parse(sid)
		if exit.Valid {
			code := int(exit.Int32)
			rec.ExitCode = &code
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// GetLogChunks returns log chunks for an execution ordered by sequence.
func (s *Store) GetLogChunks(ctx context.Context, execID uuid.UUID) ([]contracts.LogChunk, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, execution_id, stream, sequence, payload, emitted_at
		 FROM log_chunks WHERE execution_id=? ORDER BY sequence ASC`, execID.String())
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

func joinCommand(argv []string) string {
	// SQLite does not have a native array type; use JSON for safe round-tripping.
	if len(argv) == 0 {
		return "[]"
	}
	b, _ := contracts.MarshalJSON(argv)
	return string(b)
}

func splitCommand(v string) []string {
	if v == "" {
		return nil
	}
	var argv []string
	_ = contracts.UnmarshalJSON([]byte(v), &argv)
	return argv
}

func formatTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: t.UTC().Format(time.RFC3339), Valid: true}
}

func nullableInt(n *int) sql.NullInt32 {
	if n == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(*n), Valid: true}
}
