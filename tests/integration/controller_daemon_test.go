package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/Omotolani98/monocron/internal/daemon/api"
	"github.com/Omotolani98/monocron/internal/daemon/scheduler"
	"github.com/Omotolani98/monocron/internal/daemon/store"
	"github.com/Omotolani98/monocron/internal/platform/logging"
	"github.com/google/uuid"
)

func TestDaemonEndToEnd(t *testing.T) {
	log := logging.New("info")
	ctx := context.Background()

	socket := "/tmp/monocron-test-" + uuid.New().String() + ".sock"
	state := "/tmp/monocron-test-" + uuid.New().String() + ".db"
	defer os.Remove(socket)
	defer os.Remove(state)
	defer os.Remove(state + "-shm")
	defer os.Remove(state + "-wal")

	st, err := store.New(ctx, state)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	sched := scheduler.New(log, st, 1024)
	sched.Start()
	defer func() {
		stopCtx := sched.Stop()
		<-stopCtx.Done()
	}()

	srv := api.NewServer(socket, sched, st, log)
	if err := srv.Listen(); err != nil {
		t.Fatalf("listen: %v", err)
	}
	go srv.Serve()
	defer srv.Close(ctx)

	time.Sleep(100 * time.Millisecond)

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socket)
			},
		},
		Timeout: 5 * time.Second,
	}

	assignmentID := uuid.New()
	scheduleID := uuid.New()
	body, _ := json.Marshal(contracts.DaemonAssignment{
		ID:         assignmentID,
		ScheduleID: scheduleID,
		Name:       "test",
		Cron:       "*/1 * * * * *",
		Timezone:   "UTC",
		Timeout:    "5s",
		Command:    []string{"/bin/sh", "-c", "echo ok"},
		Generation: 1,
		Enabled:    true,
	})
	req, err := http.NewRequest(http.MethodPut, "http://unix/v1/assignments/"+assignmentID.String(), bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("put assignment: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	time.Sleep(2 * time.Second)

	execs, err := st.ListExecutions(ctx, assignmentID, 10)
	if err != nil {
		t.Fatalf("list executions: %v", err)
	}
	if len(execs) == 0 {
		t.Fatal("expected at least one execution")
	}
	if execs[0].State != contracts.ExecutionStateSucceeded {
		t.Fatalf("expected succeeded, got %s", execs[0].State)
	}
}
