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
	"strconv"
	"strings"
	"time"

	"github.com/Omotolani98/monocron-runner/pkg/gen"
	"github.com/Omotolani98/monocron/config"
	"github.com/google/uuid"
	yaml "sigs.k8s.io/yaml/goyaml.v2"
)

var (
	c = config.ClientConfig()
)

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

	paths := collectYamlPaths(event)
	if len(paths) == 0 {
		return &webhookResponse{Owner: owner, Repo: repo, HeadSHA: head, FilesCount: 0, Files: nil}, nil
	}

	files := make([]fetchedFile, 0, len(paths))
	for p := range paths {
		content, err := fetchRepoFile(ctx, owner, repo, head, p, event.Repository.Private)
		if err != nil || content == "" {
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

		var tf Job
		if err := yaml.Unmarshal([]byte(content), &tf); err == nil && tf.Metadata.Name != "" {
			_, _ = pushJob(ctx, tf)
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
	}
	return out
}

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

	if isPrivate {
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

func pushJob(ctx context.Context, j Job) (*gen.AddJobResponse, error) {
	t, _ := strconv.Atoi(j.Spec.Timeout)
	in := &gen.CmdJobSpec{
		JobId:          uuid.New().String(),
		Name:           j.Metadata.Name,
		Specs:          j.Spec.Schedule,
		Argv:           j.Spec.Executor.Command,
		TimeoutSeconds: int64(t),
	}
	resp, err := c.C.AddJob(ctx, in)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
