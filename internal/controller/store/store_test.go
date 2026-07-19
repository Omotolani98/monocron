package store

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/google/uuid"
)

type fakeScanner struct {
	id            uuid.UUID
	name          string
	runnerType    string
	labels        []byte
	state         string
	version       sql.NullString
	daemonVersion sql.NullString
	lastHeartbeat sql.NullTime
	createdAt     time.Time
}

func (f *fakeScanner) Scan(dest ...any) error {
	for _, d := range dest {
		switch v := d.(type) {
		case *uuid.UUID:
			*v = f.id
		case *string:
			*v = f.name
		case *contracts.RunnerType:
			*v = contracts.RunnerType(f.runnerType)
		case *[]byte:
			*v = f.labels
		case *contracts.RunnerState:
			*v = contracts.RunnerState(f.state)
		case *sql.NullString:
			if f.version.Valid {
				*v = f.version
			} else {
				*v = sql.NullString{}
			}
		case *sql.NullTime:
			if f.lastHeartbeat.Valid {
				*v = f.lastHeartbeat
			} else {
				*v = sql.NullTime{}
			}
		case *time.Time:
			*v = f.createdAt
		}
	}
	return nil
}

func TestScanRunnerHandlesNulls(t *testing.T) {
	id := uuid.New()
	labels, _ := json.Marshal(map[string]string{"zone": "home"})
	now := time.Now().UTC()
	fs := &fakeScanner{
		id:         id,
		runnerType: "bare_metal",
		labels:     labels,
		state:      string(contracts.RunnerStatePending),
		createdAt:  now,
	}

	r, err := scanRunner(fs)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if r.ID != id {
		t.Fatalf("id mismatch")
	}
	if r.Version != "" || r.DaemonVersion != "" || !r.LastHeartbeat.IsZero() {
		t.Fatalf("expected empty nullable fields, got version=%q daemon_version=%q last_heartbeat=%v", r.Version, r.DaemonVersion, r.LastHeartbeat)
	}
	if r.Labels["zone"] != "home" {
		t.Fatalf("expected label zone=home, got %v", r.Labels)
	}
}

func TestScanRunnerHandlesValues(t *testing.T) {
	id := uuid.New()
	labels, _ := json.Marshal(map[string]string{})
	now := time.Now().UTC()
	fs := &fakeScanner{
		id:            id,
		runnerType:    "vm",
		labels:        labels,
		state:         string(contracts.RunnerStateLive),
		version:       sql.NullString{String: "0.4.0", Valid: true},
		daemonVersion: sql.NullString{String: "0.4.0", Valid: true},
		lastHeartbeat: sql.NullTime{Time: now, Valid: true},
		createdAt:     now,
	}

	r, err := scanRunner(fs)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if r.Version != "0.4.0" {
		t.Fatalf("version mismatch")
	}
	if r.DaemonVersion != "0.4.0" {
		t.Fatalf("daemon version mismatch")
	}
	if !r.LastHeartbeat.Equal(now) {
		t.Fatalf("last heartbeat mismatch")
	}
}
