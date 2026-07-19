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
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeTarGz(binaryName string, content []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	w := tar.NewWriter(gz)
	hdr := &tar.Header{
		Name: binaryName,
		Mode: 0o755,
		Size: int64(len(content)),
	}
	if err := w.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := w.Write(content); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func newUpdateTestServer(t *testing.T, version string) *httptest.Server {
	t.Helper()
	osArch := "linux_amd64"
	assets := map[string][]byte{}
	checksums := &bytes.Buffer{}

	for _, binary := range updateBinaries {
		archiveName := fmt.Sprintf("%s_%s_%s.tar.gz", binary, strings.TrimPrefix(version, "v"), osArch)
		content := []byte(fmt.Sprintf("#!/bin/sh\necho %s-updated\n", binary))
		archive, err := makeTarGz(binary, content)
		if err != nil {
			t.Fatalf("make archive: %v", err)
		}
		assets[archiveName] = archive
		h := sha256.Sum256(archive)
		fmt.Fprintf(checksums, "%s  %s\n", hex.EncodeToString(h[:]), archiveName)
	}
	assets["checksums.txt"] = checksums.Bytes()

	release := releaseInfo{
		TagName: version,
	}
	for name := range assets {
		release.Assets = append(release.Assets, releaseAsset{Name: name})
	}

	mux := http.NewServeMux()
	releasePath := fmt.Sprintf("/repos/%s/%s/releases/tags/%s", updateRepoOwner, updateRepoName, version)
	mux.HandleFunc(releasePath, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(release)
	})
	mux.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/assets/")
		data, ok := assets[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(data)
	})

	server := httptest.NewServer(mux)

	// Rewrite asset URLs to point at the test server.
	for i := range release.Assets {
		release.Assets[i].BrowserDownloadURL = server.URL + "/assets/" + release.Assets[i].Name
	}

	return server
}

func TestUpdaterDryRun(t *testing.T) {
	server := newUpdateTestServer(t, "v0.0.0-test")
	defer server.Close()

	binDir := t.TempDir()
	var out bytes.Buffer
	u := updater{
		httpClient: server.Client(),
		binDir:     binDir,
		goos:       "linux",
		goarch:     "amd64",
		version:    "v0.0.0-test",
		dryRun:     true,
		baseURL:    server.URL,
		out:        &out,
	}

	if err := u.run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}

	for _, binary := range updateBinaries {
		if _, err := os.Stat(filepath.Join(binDir, binary)); !os.IsNotExist(err) {
			t.Fatalf("dry run should not install %s", binary)
		}
	}
	if !strings.Contains(out.String(), "v0.0.0-test") {
		t.Fatalf("output missing version: %s", out.String())
	}
}

func TestUpdaterInstall(t *testing.T) {
	server := newUpdateTestServer(t, "v0.0.0-test")
	defer server.Close()

	binDir := t.TempDir()
	var out bytes.Buffer
	u := updater{
		httpClient: server.Client(),
		binDir:     binDir,
		goos:       "linux",
		goarch:     "amd64",
		version:    "v0.0.0-test",
		dryRun:     false,
		baseURL:    server.URL,
		out:        &out,
	}

	if err := u.run(context.Background()); err != nil {
		t.Fatalf("run: %v\noutput:\n%s", err, out.String())
	}

	for _, binary := range updateBinaries {
		path := filepath.Join(binDir, binary)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("%s not installed: %v", binary, err)
		}
		if info.Mode().Perm()&0o111 == 0 {
			t.Fatalf("%s is not executable: %o", binary, info.Mode().Perm())
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", binary, err)
		}
		if !bytes.Contains(b, []byte(binary+"-updated")) {
			t.Fatalf("%s content unexpected: %s", binary, string(b))
		}
	}
}

func TestParseChecksums(t *testing.T) {
	input := "abc123  file.tar.gz\ndef456  other.tar.gz\n"
	got := parseChecksums(input)
	if got["file.tar.gz"] != "abc123" {
		t.Fatalf("unexpected checksum: %v", got)
	}
	if got["other.tar.gz"] != "def456" {
		t.Fatalf("unexpected checksum: %v", got)
	}
}

func TestExtractTarGzBinary(t *testing.T) {
	data, err := makeTarGz("mybin", []byte("hello"))
	if err != nil {
		t.Fatalf("make archive: %v", err)
	}
	got, err := extractTarGzBinary(data, "mybin")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !bytes.Equal(got, []byte("hello")) {
		t.Fatalf("unexpected content: %s", string(got))
	}
	if _, err := extractTarGzBinary(data, "other"); err == nil {
		t.Fatal("expected error for missing binary")
	}
}

func TestSha256Sum(t *testing.T) {
	got := sha256sum([]byte("hello"))
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Fatalf("sha256 mismatch: got %s want %s", got, want)
	}
}

// Ensure io is imported and used.
var _ = io.EOF
