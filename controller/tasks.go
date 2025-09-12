package controller

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"encore.app/controller/db"
	"encore.dev/storage/sqldb"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
)

var (
	monocronDB = sqldb.NewDatabase("monocron", sqldb.DatabaseConfig{
		Migrations: "./db/migrations",
	})

	pgxdb = sqldb.Driver[*pgxpool.Pool](monocronDB)
	q     = db.New(pgxdb)
)

type ghPushEvent struct {
	Ref        string `json:"ref"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Repository struct {
		FullName string `json:"full_name"` // "owner/repo"
		Private  bool   `json:"private"`
	} `json:"repository"`
	HeadCommit struct {
		ID string `json:"id"` // head sha
	} `json:"head_commit"`
	Commits []struct {
		Added    []string `json:"added"`
		Modified []string `json:"modified"`
		Removed  []string `json:"removed"`
	} `json:"commits"`
}

type fetchedFile struct {
	Path          string `json:"path"`
	RefSHA        string `json:"ref_sha"`
	Content       string `json:"content"`        // raw text (UTF-8)
	ContentSHA256 string `json:"content_sha256"` // hex-encoded
}

type webhookResponse struct {
	Owner      string        `json:"owner"`
	Repo       string        `json:"repo"`
	HeadSHA    string        `json:"head_sha"`
	FilesCount int           `json:"files_count"`
	Files      []fetchedFile `json:"files"`
}

//encore:api public method=POST path=/webhook
func Webhook(ctx context.Context, event *ghPushEvent) (*webhookResponse, error) {
	if event == nil {
		return nil, errors.New("invalid payload")
	}

	owner, repo := parseFullName(event.Repository.FullName)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid repository full_name: %q", event.Repository.FullName)
	}

	head := event.After
	if head == "" && event.HeadCommit.ID != "" {
		head = event.HeadCommit.ID
	}
	if head == "" {
		head = "main"
	}

	// Collect YAML paths from commits (added/modified only).
	paths := collectYamlPaths(event)
	if len(paths) == 0 {
		// Nothing to fetch; return minimal response.
		return &webhookResponse{Owner: owner, Repo: repo, HeadSHA: head, FilesCount: 0, Files: nil}, nil
	}

	files := make([]fetchedFile, 0, len(paths))
	for p := range paths {
		content, err := fetchRepoFile(ctx, owner, repo, head, p, event.Repository.Private)
		if err != nil || content == "" {
			// Skip files that can't be fetched (deleted or not present at head).
			continue
		}
		sum := sha256.Sum256([]byte(content))
		ff := fetchedFile{
			Path:          p,
			RefSHA:        head,
			Content:       content,
			ContentSHA256: hex.EncodeToString(sum[:]),
		}
		files = append(files, ff)

		// Parse YAML into TaskFile and persist + enqueue runs.
		var tf TaskFile
		if err := yaml.Unmarshal([]byte(content), &tf); err == nil && tf.Metadata.Name != "" {
			_ = upsertAndPlan(ctx, tf)
		}
	}

	resp := &webhookResponse{
		Owner:      owner,
		Repo:       repo,
		HeadSHA:    head,
		FilesCount: len(files),
		Files:      files,
	}
	return resp, nil
}

func parseFullName(full string) (owner, repo string) {
	parts := strings.Split(full, "/")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

func collectYamlPaths(e *ghPushEvent) map[string]struct{} {
	out := make(map[string]struct{})
	add := func(p string) {
		ext := strings.ToLower(filepath.Ext(p))
		if ext == ".yaml" || ext == ".yml" {
			out[p] = struct{}{}
		}
	}
	for _, c := range e.Commits {
		for _, p := range c.Added {
			add(p)
		}
		for _, p := range c.Modified {
			add(p)
		}
		// removed files are ignored since they cannot be fetched at head if deleted
	}
	return out
}

// fetchRepoFile fetches the file content at given ref. It supports both public and private repos.
// If GITHUB_TOKEN is set, it uses the GitHub API which works for private repos.
func fetchRepoFile(ctx context.Context, owner, repo, ref, path string, isPrivate bool) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))

	if token != "" {
		// Use GitHub Contents API: returns base64 content when using API.
		url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, ref)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			// Not found or no access
			return "", fmt.Errorf("github api status: %d", resp.StatusCode)
		}
		var body struct {
			Encoding string `json:"encoding"`
			Content  string `json:"content"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return "", err
		}
		if body.Encoding == "base64" {
			dec, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(body.Content, "\n", ""))
			if err != nil {
				return "", err
			}
			return string(dec), nil
		}
		return body.Content, nil
	}

	// Fallback to raw URL for public repos.
	if isPrivate {
		// Cannot fetch private repo without token.
		return "", errors.New("private repo requires GITHUB_TOKEN")
	}
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, ref, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("raw content status: %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ---- Task YAML, planning, and persistence helpers ----

// TaskFile mirrors the expected YAML schema.
type TaskFile struct {
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`
	Kind       string `yaml:"kind" json:"kind"`
	Metadata   struct {
		Name string `yaml:"name" json:"name"`
	} `yaml:"metadata" json:"metadata"`
	Spec struct {
		Schedule          string `yaml:"schedule" json:"schedule"`
		Timezone          string `yaml:"timezone" json:"timezone"`
		ConcurrencyPolicy string `yaml:"concurrencyPolicy" json:"concurrencyPolicy"`
		Retry             struct {
			MaxAttempts int     `yaml:"maxAttempts" json:"maxAttempts"`
			Backoff     *string `yaml:"backoff" json:"backoff"`
		} `yaml:"retry" json:"retry"`
		Catchup       bool   `yaml:"catchup" json:"catchup"`
		CatchupWindow string `yaml:"catchupWindow" json:"catchupWindow"`
		Executor      any    `yaml:"executor" json:"executor"`
	} `yaml:"spec" json:"spec"`
}

type runCandidate struct {
	ScheduledAt time.Time
	Source      string // "regular" | "catchup"
}

// upsertAndPlan upserts the task and enqueues runs if a DB is available.
func upsertAndPlan(ctx context.Context, tf TaskFile) error {
	// Prepare UpsertTask params.
	catchupWindow := time.Duration(0)
	if tf.Spec.CatchupWindow != "" {
		if d, err := time.ParseDuration(tf.Spec.CatchupWindow); err == nil {
			catchupWindow = d
		}
	}
	execJSON, _ := json.Marshal(tf.Spec.Executor)

	t, err := q.UpsertTask(ctx, QueriesUpsertParams(tf, catchupWindow, execJSON))
	if err != nil {
		return err
	}

	// Plan runs (next + catchup).
	var lastPtr *time.Time
	if last, err := q.GetLastScheduledForTask(ctx, t.ID); err == nil && !last.IsZero() {
		lastPtr = &last
	}

	plan, err := planRuns(tf, time.Now(), lastPtr)
	if err != nil {
		return err
	}

	// Enqueue runs.
	for _, rc := range plan {
		_, _ = q.EnqueueRun(ctx, EnqueueParams(t.ID, rc))
	}
	return nil
}

// QueriesUpsertParams converts TaskFile to db.UpsertTaskParams.
func QueriesUpsertParams(tf TaskFile, catchupWindow time.Duration, execJSON []byte) db.UpsertTaskParams {
	return db.UpsertTaskParams{
		Name:              tf.Metadata.Name,
		Schedule:          tf.Spec.Schedule,
		Timezone:          tf.Spec.Timezone,
		ConcurrencyPolicy: zerodef(tf.Spec.ConcurrencyPolicy, "Allow"),
		Retries:           tf.Spec.Retry.MaxAttempts,
		Backoff:           tf.Spec.Retry.Backoff,
		Catchup:           tf.Spec.Catchup,
		CatchupWindow:     catchupWindow,
		Executor:          execJSON,
	}
}

// planRuns computes the next run and catchup runs from last scheduled.
func planRuns(tf TaskFile, now time.Time, lastScheduled *time.Time) ([]runCandidate, error) {
	spec := tf.Spec

	loc := time.UTC
	if spec.Timezone != "" {
		if l, err := time.LoadLocation(spec.Timezone); err == nil {
			loc = l
		}
	}
	sch, err := cron.ParseStandard(spec.Schedule)
	if err != nil {
		return nil, fmt.Errorf("invalid cron %q: %v", spec.Schedule, err)
	}
	nowLocal := now.In(loc)

	// Next run
	nextLocal := sch.Next(nowLocal)
	runs := []runCandidate{{ScheduledAt: nextLocal.UTC(), Source: "regular"}}

	if !spec.Catchup {
		return runs, nil
	}

	var window time.Duration
	if spec.CatchupWindow != "" {
		if d, err := time.ParseDuration(spec.CatchupWindow); err == nil {
			window = d
		}
	}
	startLocal := nowLocal.Add(-time.Second)
	if lastScheduled != nil {
		startLocal = lastScheduled.In(loc)
	}
	if window > 0 {
		minLocal := nowLocal.Add(-window)
		if startLocal.Before(minLocal) {
			startLocal = minLocal
		}
	}
	iter := startLocal
	for i := 0; i < 10000; i++ { // safety cap
		fired := sch.Next(iter)
		if !fired.Before(nowLocal) {
			break
		}
		runs = append(runs, runCandidate{ScheduledAt: fired.UTC(), Source: "catchup"})
		iter = fired
	}
	return runs, nil
}

// EnqueueParams maps runCandidate to db.EnqueueRunParams.
func EnqueueParams(taskID uuid.UUID, rc runCandidate) db.EnqueueRunParams {
	return db.EnqueueRunParams{TaskID: taskID, ScheduledAt: rc.ScheduledAt, Status: "QUEUED", Source: rc.Source}
}

// // getQueries returns a DB query interface if configured. Placeholder for integration.
// func getQueries() *db.Queries { return nil }

func zerodef[T ~string](v T, def T) T {
	if v == "" {
		return def
	}
	return v
}
