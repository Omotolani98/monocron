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
		files = append(files, fetchedFile{
			Path:          p,
			RefSHA:        head,
			Content:       content,
			ContentSHA256: hex.EncodeToString(sum[:]),
		})
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
