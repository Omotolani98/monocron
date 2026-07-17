package store

import (
	"context"
	"testing"
	"time"

	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/Omotolani98/monocron/internal/daemon/scheduler"
	"github.com/google/uuid"
)

func TestStoreAssignmentRoundTrip(t *testing.T) {
	ctx := context.Background()
	s, err := New(ctx, ":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	a := scheduler.Assignment{
		ID:         uuid.New(),
		ScheduleID: uuid.New(),
		Name:       "test",
		Cron:       "*/5 * * * *",
		Timezone:   "UTC",
		Timeout:    10 * time.Second,
		Command:    []string{"echo", "hello"},
		Generation: 1,
		Enabled:    true,
	}
	if err := s.SaveAssignment(ctx, a); err != nil {
		t.Fatalf("save assignment: %v", err)
	}

	loaded, err := s.LoadAssignments(ctx)
	if err != nil {
		t.Fatalf("load assignments: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(loaded))
	}
	if loaded[0].ID != a.ID || loaded[0].Name != a.Name {
		t.Fatalf("assignment mismatch: %+v", loaded[0])
	}
}

func TestStoreExecutionRoundTrip(t *testing.T) {
	ctx := context.Background()
	s, err := New(ctx, ":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	rec := scheduler.ExecutionRecord{
		ID:           uuid.New(),
		AssignmentID: uuid.New(),
		ScheduleID:   uuid.New(),
		State:        contracts.ExecutionStateSucceeded,
		Stdout:       []byte("hello"),
	}
	if err := s.PersistExecution(ctx, rec); err != nil {
		t.Fatalf("persist execution: %v", err)
	}

	got, err := s.GetExecution(ctx, rec.ID)
	if err != nil {
		t.Fatalf("get execution: %v", err)
	}
	if got.State != rec.State || string(got.Stdout) != string(rec.Stdout) {
		t.Fatalf("execution mismatch: %+v", got)
	}
}
