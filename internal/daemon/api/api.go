package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/Omotolani98/monocron/internal/daemon/scheduler"
	"github.com/Omotolani98/monocron/internal/daemon/store"
	"github.com/Omotolani98/monocron/internal/platform/version"
	"github.com/google/uuid"
)

// Server hosts the daemon Unix-socket HTTP API.
type Server struct {
	socketPath string
	scheduler  *scheduler.Scheduler
	store      *store.Store
	log        *slog.Logger
	srv        *http.Server
	listener   net.Listener
}

// NewServer creates a daemon API server.
func NewServer(socketPath string, sched *scheduler.Scheduler, st *store.Store, log *slog.Logger) *Server {
	mux := http.NewServeMux()
	s := &Server{
		socketPath: socketPath,
		scheduler:  sched,
		store:      st,
		log:        log,
		srv: &http.Server{
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	}
	mux.HandleFunc("GET /v1/health", s.handleHealth)
	mux.HandleFunc("PUT /v1/assignments/{id}", s.handlePutAssignment)
	mux.HandleFunc("GET /v1/assignments", s.handleListAssignments)
	mux.HandleFunc("GET /v1/assignments/{id}", s.handleGetAssignment)
	mux.HandleFunc("DELETE /v1/assignments/{id}", s.handleDeleteAssignment)
	mux.HandleFunc("GET /v1/assignments/{id}/executions", s.handleListExecutions)
	mux.HandleFunc("GET /v1/executions/{id}", s.handleGetExecution)
	mux.HandleFunc("GET /v1/executions/{id}/logs", s.handleGetExecutionLogs)
	mux.HandleFunc("POST /v1/executions/{id}/cancel", s.handleCancelExecution)
	mux.HandleFunc("GET /v1/events", s.handleEvents)
	mux.HandleFunc("POST /v1/shutdown", s.handleShutdown)
	return s
}

// Listen creates the Unix socket listener, cleaning up stale sockets.
func (s *Server) Listen() error {
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0o755); err != nil {
		return fmt.Errorf("create socket directory: %w", err)
	}
	_ = os.Remove(s.socketPath)
	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on unix socket: %w", err)
	}
	if err := os.Chmod(s.socketPath, 0o660); err != nil {
		_ = ln.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}
	s.listener = ln
	return nil
}

// Serve accepts connections on the prepared listener.
func (s *Server) Serve() error {
	if s.listener == nil {
		return errors.New("server not listening")
	}
	s.log.Info("daemon listening", "socket", s.socketPath)
	return s.srv.Serve(s.listener)
}

// Close gracefully shuts down the server and removes the socket.
func (s *Server) Close(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	err := s.srv.Shutdown(ctx)
	if s.listener != nil {
		_ = s.listener.Close()
	}
	_ = os.Remove(s.socketPath)
	return err
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": version.Version,
	})
}

func (s *Server) handlePutAssignment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid assignment id")
		return
	}
	var da contracts.DaemonAssignment
	if err := decodeJSON(r.Body, &da); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if da.ID != id {
		writeError(w, http.StatusBadRequest, "assignment id mismatch")
		return
	}
	timeout, err := time.ParseDuration(da.Timeout)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid timeout")
		return
	}
	if len(da.Command) == 0 || strings.TrimSpace(da.Command[0]) == "" {
		writeError(w, http.StatusBadRequest, "empty command")
		return
	}

	a := scheduler.Assignment{
		ID:         da.ID,
		ScheduleID: da.ScheduleID,
		Name:       da.Name,
		Cron:       da.Cron,
		Timezone:   da.Timezone,
		Timeout:    timeout,
		Command:    da.Command,
		Generation: da.Generation,
		Enabled:    da.Enabled,
	}

	if err := s.scheduler.Add(r.Context(), a); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.store.SaveAssignment(r.Context(), a); err != nil {
		s.log.Error("save assignment", "error", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "applied"})
}

func (s *Server) handleListAssignments(w http.ResponseWriter, r *http.Request) {
	items := s.scheduler.List()
	out := make([]contracts.DaemonAssignment, 0, len(items))
	for _, a := range items {
		out = append(out, toDaemonAssignment(a))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetAssignment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid assignment id")
		return
	}
	a, ok := s.scheduler.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "assignment not found")
		return
	}
	writeJSON(w, http.StatusOK, toDaemonAssignment(a))
}

func (s *Server) handleDeleteAssignment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid assignment id")
		return
	}
	if err := s.scheduler.Remove(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.store.DeleteAssignment(r.Context(), id); err != nil {
		s.log.Error("delete assignment", "error", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (s *Server) handleListExecutions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid assignment id")
		return
	}
	recs, err := s.store.ListExecutions(r.Context(), id, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]contracts.DaemonExecutionResult, 0, len(recs))
	for _, rec := range recs {
		out = append(out, toDaemonResult(rec))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetExecution(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid execution id")
		return
	}
	rec, err := s.store.GetExecution(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "execution not found")
		return
	}
	writeJSON(w, http.StatusOK, toDaemonResult(rec))
}

func (s *Server) handleGetExecutionLogs(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid execution id")
		return
	}
	chunks, err := s.store.GetLogChunks(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, chunks)
}

func (s *Server) handleCancelExecution(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid execution id")
		return
	}
	if !s.scheduler.Cancel(id) {
		writeError(w, http.StatusNotFound, "execution not found or not running")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancel requested"})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	// The store is the durable source of events for the runner to poll.
	// For the MVP we return empty; runners will also poll executions directly.
	writeJSON(w, http.StatusOK, contracts.DaemonEventsResponse{Events: []contracts.DaemonEvent{}})
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "shutting down"})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Close(ctx)
	}()
}

func toDaemonAssignment(a scheduler.Assignment) contracts.DaemonAssignment {
	return contracts.DaemonAssignment{
		ID:         a.ID,
		ScheduleID: a.ScheduleID,
		Name:       a.Name,
		Cron:       a.Cron,
		Timezone:   a.Timezone,
		Timeout:    a.Timeout.String(),
		Command:    a.Command,
		Generation: a.Generation,
		Enabled:    a.Enabled,
	}
}

func toDaemonResult(rec scheduler.ExecutionRecord) contracts.DaemonExecutionResult {
	return contracts.DaemonExecutionResult{
		ID:           rec.ID,
		AssignmentID: rec.AssignmentID,
		State:        rec.State,
		StartedAt:    rec.StartedAt,
		FinishedAt:   rec.FinishedAt,
		ExitCode:     rec.ExitCode,
		ErrorMessage: rec.ErrorMessage,
	}
}

func decodeJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, contracts.ErrorResponse{Code: http.StatusText(status), Message: message})
}
