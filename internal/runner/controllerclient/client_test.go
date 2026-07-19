package controllerclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Omotolani98/monocron/internal/contracts"
)

func TestPollAssignmentsURL(t *testing.T) {
	var gotPath string
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"generation":0,"assignments":[]}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.PollAssignments(context.Background(), 7)
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if gotPath != "/api/v1/runners/assignments" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotQuery != "after=7" {
		t.Fatalf("unexpected query: %s", gotQuery)
	}
}

func TestRequestIncludesAuthorization(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"generation":0,"assignments":[]}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "my-token")
	_, _ = c.PollAssignments(context.Background(), 0)
	if auth != "Bearer my-token" {
		t.Fatalf("unexpected auth header: %s", auth)
	}
}

func TestPollAssignmentsHandlesServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `unauthorized`)
	}))
	defer srv.Close()

	c := New(srv.URL, "bad-token")
	_, err := c.PollAssignments(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "http 401: unauthorized" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHeartbeatURL(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"state":"live"}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "token")
	_, err := c.Heartbeat(context.Background(), contracts.HeartbeatRequest{})
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if gotPath != "/api/v1/runners/heartbeat" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
}

func TestBaseURLTrimmed(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"generation":0,"assignments":[]}`)
	}))
	defer srv.Close()

	c := New(srv.URL+"/", "token")
	_, _ = c.PollAssignments(context.Background(), 0)
	if gotPath != "/api/v1/runners/assignments" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
}

