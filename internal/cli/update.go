package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	updateRepoOwner = "Omotolani98"
	updateRepoName  = "monocron"
)

var updateBinaries = []string{"monocronctl", "monocron-controller", "monocron-runner", "monocrond"}

// releaseAsset is a subset of the GitHub release asset JSON.
type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// releaseInfo is a subset of the GitHub release JSON.
type releaseInfo struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

func newUpdateCmd() *cobra.Command {
	var binDir string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "update [version]",
		Short: "Update all Monocron binaries",
		Long:  "Update all Monocron binaries to the specified release version (or latest if omitted).",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := ""
			if len(args) > 0 {
				version = args[0]
			}
			if binDir == "" {
				exe, err := os.Executable()
				if err != nil {
					return fmt.Errorf("locate current binary: %w", err)
				}
				binDir = filepath.Dir(exe)
			}
			u := updater{
				httpClient: &http.Client{Timeout: 5 * time.Minute},
				binDir:     binDir,
				goos:       runtime.GOOS,
				goarch:     runtime.GOARCH,
				version:    version,
				dryRun:     dryRun,
				out:        cmd.OutOrStdout(),
			}
			return u.run(cmd.Context())
		},
	}
	cmd.Flags().StringVar(&binDir, "bin-dir", "", "directory containing Monocron binaries (default: directory of monocronctl)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would be updated without changing anything")
	return cmd
}

type updater struct {
	httpClient *http.Client
	binDir     string
	goos       string
	goarch     string
	version    string
	dryRun     bool
	baseURL    string
	out        io.Writer
}

func (u *updater) run(ctx context.Context) error {
	if u.goos != "linux" && u.goos != "darwin" {
		return fmt.Errorf("updates are only supported on linux and darwin (current: %s)", u.goos)
	}

	tag, err := u.resolveTag(ctx)
	if err != nil {
		return err
	}

	release, err := u.fetchRelease(ctx, tag)
	if err != nil {
		return err
	}

	checksums, err := u.fetchChecksums(ctx, release)
	if err != nil {
		return err
	}

	fmt.Fprintf(u.out, "Updating Monocron binaries to %s for %s/%s...\n", release.TagName, u.goos, u.goarch)

	for _, binary := range updateBinaries {
		if err := u.updateBinary(ctx, binary, release, checksums); err != nil {
			return fmt.Errorf("update %s: %w", binary, err)
		}
	}

	fmt.Fprintln(u.out, "All binaries updated.")
	return nil
}

func (u *updater) resolveTag(ctx context.Context) (string, error) {
	if u.version == "" {
		release, err := u.fetchLatestRelease(ctx)
		if err != nil {
			return "", err
		}
		return release.TagName, nil
	}
	tag := u.version
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return tag, nil
}

func (u *updater) updateBinary(ctx context.Context, binary string, release releaseInfo, checksums map[string]string) error {
	archiveName := fmt.Sprintf("%s_%s_%s_%s.tar.gz", binary, strings.TrimPrefix(release.TagName, "v"), u.goos, u.goarch)
	asset, err := u.findAsset(release, archiveName)
	if err != nil {
		return err
	}

	fmt.Fprintf(u.out, "  %s -> %s\n", binary, asset.BrowserDownloadURL)
	if u.dryRun {
		return nil
	}

	archiveData, err := u.download(ctx, asset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}

	expected, ok := checksums[archiveName]
	if !ok {
		return fmt.Errorf("missing checksum for %s", archiveName)
	}
	if got := sha256sum(archiveData); got != expected {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", archiveName, got, expected)
	}

	binaryData, err := extractTarGzBinary(archiveData, binary)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	target := filepath.Join(u.binDir, binary)
	if err := u.install(target, binaryData); err != nil {
		return fmt.Errorf("install: %w", err)
	}
	return nil
}

func (u *updater) install(target string, data []byte) error {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp := target + ".new"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return err
	}

	backup := target + ".bak"
	if _, err := os.Stat(target); err == nil {
		if err := os.Rename(target, backup); err != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("backup existing binary: %w", err)
		}
	}

	if err := os.Rename(tmp, target); err != nil {
		_ = os.Rename(backup, target)
		return err
	}

	_ = os.Remove(backup)
	return nil
}

func (u *updater) apiURL(path string) string {
	base := u.baseURL
	if base == "" {
		base = "https://api.github.com"
	}
	return base + path
}

func (u *updater) fetchLatestRelease(ctx context.Context) (releaseInfo, error) {
	url := u.apiURL(fmt.Sprintf("/repos/%s/%s/releases/latest", updateRepoOwner, updateRepoName))
	return u.fetchReleaseJSON(ctx, url)
}

func (u *updater) fetchRelease(ctx context.Context, tag string) (releaseInfo, error) {
	url := u.apiURL(fmt.Sprintf("/repos/%s/%s/releases/tags/%s", updateRepoOwner, updateRepoName, tag))
	return u.fetchReleaseJSON(ctx, url)
}

func (u *updater) fetchReleaseJSON(ctx context.Context, url string) (releaseInfo, error) {
	var release releaseInfo
	data, err := u.download(ctx, url)
	if err != nil {
		return release, fmt.Errorf("fetch release: %w", err)
	}
	if err := json.Unmarshal(data, &release); err != nil {
		return release, fmt.Errorf("parse release: %w", err)
	}
	if release.TagName == "" {
		return release, fmt.Errorf("release not found at %s", url)
	}
	return release, nil
}

func (u *updater) fetchChecksums(ctx context.Context, release releaseInfo) (map[string]string, error) {
	asset, err := u.findChecksumsAsset(release)
	if err != nil {
		return nil, err
	}
	data, err := u.download(ctx, asset.BrowserDownloadURL)
	if err != nil {
		return nil, fmt.Errorf("download checksums: %w", err)
	}
	return parseChecksums(string(data)), nil
}

func (u *updater) findChecksumsAsset(release releaseInfo) (releaseAsset, error) {
	for _, a := range release.Assets {
		if a.Name == "checksums.txt" || strings.HasSuffix(a.Name, "_checksums.txt") {
			return a, nil
		}
	}
	return releaseAsset{}, fmt.Errorf("asset checksums.txt not found in release %s", release.TagName)
}

func (u *updater) findAsset(release releaseInfo, name string) (releaseAsset, error) {
	for _, a := range release.Assets {
		if a.Name == name {
			return a, nil
		}
	}
	return releaseAsset{}, fmt.Errorf("asset %s not found in release %s", name, release.TagName)
}

func (u *updater) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream, application/json")
	req.Header.Set("User-Agent", "monocronctl")
	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 100<<20))
}

func parseChecksums(text string) map[string]string {
	checksums := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := parts[1]
		if strings.HasPrefix(name, "*") {
			name = name[1:]
		}
		checksums[name] = parts[0]
	}
	return checksums
}

func sha256sum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func extractTarGzBinary(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if h.FileInfo().Mode().IsRegular() && filepath.Base(h.Name) == name {
			return io.ReadAll(io.LimitReader(tr, 100<<20))
		}
	}
	return nil, fmt.Errorf("binary %q not found in archive", name)
}
