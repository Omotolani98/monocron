package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/Omotolani98/monocron/internal/controller/placement"
	"github.com/Omotolani98/monocron/internal/controller/store"
	"github.com/Omotolani98/monocron/internal/platform/version"
	"github.com/google/uuid"
)

// Server hosts the controller REST API.
type Server struct {
	db  *store.DB
	log *slog.Logger
	srv *http.Server
}

// NewServer creates a controller API server.
func NewServer(addr string, db *store.DB, log *slog.Logger) *Server {
	mux := http.NewServeMux()
	s := &Server{
		db:  db,
		log: log,
		srv: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	}

	// Public controller API
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)

	mux.HandleFunc("POST /api/v1/enrollment-tokens", s.requireAuth(s.handleCreateEnrollmentToken))
	mux.HandleFunc("POST /api/v1/runners/enroll", s.handleEnroll)
	mux.HandleFunc("POST /api/v1/runners/heartbeat", s.requireRunnerAuth(s.handleHeartbeat))
	mux.HandleFunc("GET /api/v1/runners/assignments", s.requireRunnerAuth(s.handlePollAssignments))
	mux.HandleFunc("POST /api/v1/runners/observations", s.requireRunnerAuth(s.handleSubmitObservations))
	mux.HandleFunc("POST /api/v1/runners/execution-events", s.requireRunnerAuth(s.handleSubmitExecutionEvents))
	mux.HandleFunc("POST /api/v1/runners/log-chunks", s.requireRunnerAuth(s.handleSubmitLogChunks))

	mux.HandleFunc("GET /api/v1/runners", s.requireAuth(s.handleListRunners))
	mux.HandleFunc("GET /api/v1/runners/{id}", s.requireAuth(s.handleGetRunner))
	mux.HandleFunc("POST /api/v1/runners/{id}/drain", s.requireAuth(s.handleDrainRunner))
	mux.HandleFunc("DELETE /api/v1/runners/{id}", s.requireAuth(s.handleDeleteRunner))

	mux.HandleFunc("POST /api/v1/schedules", s.requireAuth(s.handleCreateSchedule))
	mux.HandleFunc("GET /api/v1/schedules", s.requireAuth(s.handleListSchedules))
	mux.HandleFunc("GET /api/v1/schedules/{id}", s.requireAuth(s.handleGetSchedule))
	mux.HandleFunc("PUT /api/v1/schedules/{id}", s.requireAuth(s.handleUpdateSchedule))
	mux.HandleFunc("DELETE /api/v1/schedules/{id}", s.requireAuth(s.handleDeleteSchedule))

	mux.HandleFunc("GET /api/v1/executions", s.requireAuth(s.handleListExecutions))
	mux.HandleFunc("GET /api/v1/executions/{id}", s.requireAuth(s.handleGetExecution))
	mux.HandleFunc("POST /api/v1/executions/{id}/cancel", s.requireAuth(s.handleCancelExecution))
	mux.HandleFunc("GET /api/v1/executions/{id}/logs", s.requireAuth(s.handleGetExecutionLogs))

	mux.HandleFunc("GET /api/v1/audit-events", s.requireAuth(s.handleListAuditEvents))

	return s
}

// Run starts the controller server.
func (s *Server) Run() error {
	s.log.Info("controller listening", "addr", s.srv.Addr)
	return s.srv.ListenAndServe()
}

// Close shuts down the server.
func (s *Server) Close(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version.Version})
}

func (s *Server) handleCreateEnrollmentToken(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreateEnrollmentTokenRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	plaintext, resp, err := s.db.CreateEnrollmentToken(r.Context(), req)
	if err != nil {
		s.log.Error("create enrollment token", "error", err)
		writeError(w, http.StatusInternalServerError, "could not create token")
		return
	}
	_ = s.db.Audit(r.Context(), actor(r), "enrollment_token.create", "token:"+resp.Token, "created enrollment token", map[string]any{"scope": req.Scope})
	// Return the plaintext token only once.
	writeJSON(w, http.StatusCreated, contracts.CreateEnrollmentTokenResponse{Token: plaintext, ExpiresAt: resp.ExpiresAt})
}

func (s *Server) handleEnroll(w http.ResponseWriter, r *http.Request) {
	var req contracts.EnrollRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	runner, accessToken, err := s.db.EnrollRunner(r.Context(), req.Token)
	if err != nil {
		s.log.Error("enroll runner", "error", err)
		status := http.StatusInternalServerError
		msg := "could not create runner"
		if errors.Is(err, sql.ErrNoRows) || isTokenError(err) {
			status = http.StatusUnauthorized
			msg = err.Error()
		}
		writeError(w, status, msg)
		return
	}
	writeJSON(w, http.StatusCreated, contracts.EnrollResponse{RunnerID: runner.ID, AccessToken: accessToken})
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	runnerID := runnerIDFromContext(r.Context())
	var req contracts.HeartbeatRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.db.UpdateRunnerHeartbeat(r.Context(), runnerID, req); err != nil {
		s.log.Error("heartbeat", "error", err)
		writeError(w, http.StatusInternalServerError, "could not update heartbeat")
		return
	}
	runner, err := s.db.GetRunner(r.Context(), runnerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, contracts.HeartbeatResponse{State: runner.State})
}

func (s *Server) handlePollAssignments(w http.ResponseWriter, r *http.Request) {
	runnerID := runnerIDFromContext(r.Context())
	after := int64(0)
	if v := r.URL.Query().Get("after"); v != "" {
		fmt.Sscanf(v, "%d", &after)
	}
	assignments, err := s.db.ListAssignmentsForRunner(r.Context(), runnerID, after)
	if err != nil {
		s.log.Error("list assignments", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var generation int64
	for _, a := range assignments {
		if a.Generation > generation {
			generation = a.Generation
		}
	}
	writeJSON(w, http.StatusOK, contracts.PollAssignmentsResponse{Generation: generation, Assignments: assignments})
}

func (s *Server) handleSubmitObservations(w http.ResponseWriter, r *http.Request) {
	var req contracts.SubmitObservationsRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	for _, obs := range req.Observations {
		if err := s.db.UpdateAssignmentState(r.Context(), obs.AssignmentID, obs.State); err != nil {
			s.log.Error("update assignment state", "error", err)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleSubmitExecutionEvents(w http.ResponseWriter, r *http.Request) {
	runnerID := runnerIDFromContext(r.Context())
	var req contracts.SubmitExecutionEventsRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	for _, ev := range req.Events {
		execution, err := s.db.GetExecution(r.Context(), ev.ExecutionID)
		if err != nil {
			// Create execution placeholder if it does not exist.
			execution, err = s.db.CreateExecution(r.Context(), ev.AssignmentID, uuid.Nil, runnerID)
			if err != nil {
				s.log.Error("create execution", "error", err)
				continue
			}
		}
		var started, finished *time.Time
		if ev.State == contracts.ExecutionStateRunning {
			started = &ev.EmittedAt
		}
		if isTerminal(ev.State) {
			finished = &ev.EmittedAt
		}
		exitCode := execution.ExitCode
		if err := s.db.UpdateExecutionState(r.Context(), execution.ID, ev.State, started, finished, exitCode, ev.Message); err != nil {
			s.log.Error("update execution state", "error", err)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleSubmitLogChunks(w http.ResponseWriter, r *http.Request) {
	var req contracts.SubmitLogChunksRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.db.SaveLogChunks(r.Context(), req.Chunks); err != nil {
		s.log.Error("save log chunks", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListRunners(w http.ResponseWriter, r *http.Request) {
	items, next, err := s.db.ListRunners(r.Context(), r.URL.Query().Get("cursor"), parseLimit(r.URL.Query().Get("limit")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, contracts.ListRunnersResponse{Items: items, NextCursor: next})
}

func (s *Server) handleGetRunner(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid runner id")
		return
	}
	runner, err := s.db.GetRunner(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, runner)
}

func (s *Server) handleDrainRunner(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid runner id")
		return
	}
	if err := s.db.UpdateRunnerState(r.Context(), id, contracts.RunnerStateDraining); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.db.Audit(r.Context(), actor(r), "runner.drain", id.String(), "", nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "draining"})
}

func (s *Server) handleDeleteRunner(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid runner id")
		return
	}
	if err := s.db.UpdateRunnerState(r.Context(), id, contracts.RunnerStateRevoked); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.db.Audit(r.Context(), actor(r), "runner.delete", id.String(), "", nil)
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreateScheduleRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name == "" || req.Cron == "" || len(req.Command) == 0 {
		writeError(w, http.StatusBadRequest, "name, cron, and command are required")
		return
	}
	schedule, err := s.db.CreateSchedule(r.Context(), req)
	if err != nil {
		s.log.Error("create schedule", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.db.Audit(r.Context(), actor(r), "schedule.create", schedule.ID.String(), "", req)

	// Place schedule on matching runners.
	if err := s.placeSchedule(r.Context(), schedule); err != nil {
		s.log.Error("place schedule", "error", err)
	}

	writeJSON(w, http.StatusCreated, schedule)
}

func (s *Server) placeSchedule(ctx context.Context, schedule contracts.ScheduleSpec) error {
	runners, _, err := s.db.ListRunners(ctx, "", 1000)
	if err != nil {
		return err
	}
	matched := placement.SelectRunners(runners, schedule.Labels)
	if len(matched) == 0 {
		return nil
	}
	// Use the current max generation + 1
	generation := time.Now().UnixNano()
	for _, r := range matched {
		if _, err := s.db.CreateAssignment(ctx, schedule.ID, r.ID, generation); err != nil {
			s.log.Error("create assignment", "runner_id", r.ID, "error", err)
		}
	}
	return nil
}

func (s *Server) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	items, next, err := s.db.ListSchedules(r.Context(), r.URL.Query().Get("cursor"), parseLimit(r.URL.Query().Get("limit")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, contracts.ListSchedulesResponse{Items: items, NextCursor: next})
}

func (s *Server) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid schedule id")
		return
	}
	schedule, err := s.db.GetSchedule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, schedule)
}

func (s *Server) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid schedule id")
		return
	}
	var req contracts.UpdateScheduleRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	schedule, err := s.db.UpdateSchedule(r.Context(), id, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.db.Audit(r.Context(), actor(r), "schedule.update", id.String(), "", req)
	// Re-place if labels changed
	if req.Labels != nil {
		_ = s.placeSchedule(r.Context(), schedule)
	}
	writeJSON(w, http.StatusOK, schedule)
}

func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid schedule id")
		return
	}
	if err := s.db.DeleteSchedule(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.db.Audit(r.Context(), actor(r), "schedule.delete", id.String(), "", nil)
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleListExecutions(w http.ResponseWriter, r *http.Request) {
	items, next, err := s.db.ListExecutions(r.Context(), r.URL.Query().Get("cursor"), parseLimit(r.URL.Query().Get("limit")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, contracts.ListExecutionsResponse{Items: items, NextCursor: next})
}

func (s *Server) handleGetExecution(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid execution id")
		return
	}
	execution, err := s.db.GetExecution(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, execution)
}

func (s *Server) handleCancelExecution(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid execution id")
		return
	}
	if err := s.db.UpdateExecutionState(r.Context(), id, contracts.ExecutionStateCanceled, nil, &time.Time{}, nil, "canceled by user"); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.db.Audit(r.Context(), actor(r), "execution.cancel", id.String(), "", nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "canceled"})
}

func (s *Server) handleGetExecutionLogs(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid execution id")
		return
	}
	chunks, err := s.db.GetExecutionLogChunks(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, chunks)
}

func (s *Server) handleListAuditEvents(w http.ResponseWriter, r *http.Request) {
	items, next, err := s.db.ListAuditEvents(r.Context(), r.URL.Query().Get("cursor"), parseLimit(r.URL.Query().Get("limit")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, contracts.ListAuditEventsResponse{Items: items, NextCursor: next})
}

// requireAuth is a placeholder for admin authentication. In production validate bearer tokens or mTLS.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: implement real admin auth.
		next(w, r)
	}
}

// requireRunnerAuth validates runner access tokens and injects runner ID into context.
func (s *Server) requireRunnerAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization")
			return
		}
		runnerID, err := s.db.ValidateAccessToken(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), runnerIDKey{}, runnerID)))
	}
}

type runnerIDKey struct{}

func runnerIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(runnerIDKey{}).(uuid.UUID)
	return id
}

func isTerminal(state contracts.ExecutionState) bool {
	switch state {
	case contracts.ExecutionStateSucceeded, contracts.ExecutionStateFailed, contracts.ExecutionStateTimedOut, contracts.ExecutionStateCanceled:
		return true
	}
	return false
}

func decodeJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

func parseLimit(v string) int {
	var n int
	fmt.Sscanf(v, "%d", &n)
	return n
}

func actor(r *http.Request) string {
	return r.Header.Get("X-Actor")
}

// isTokenError reports whether err is an enrollment-token validation error.
func isTokenError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "invalid or expired token"):
		return true
	case strings.Contains(msg, "token already consumed"):
		return true
	case strings.Contains(msg, "token expired"):
		return true
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, contracts.ErrorResponse{Code: http.StatusText(status), Message: message})
}
