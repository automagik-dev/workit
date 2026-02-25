package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUpdateCmdRun_NoOpRejected(t *testing.T) {
	t.Parallel()

	cmd := &UpdateCmd{
		SkipBinary: true,
		SkipSkills: true,
	}
	err := cmd.Run(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "nothing to do") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGithubTokenPriority(t *testing.T) {
	t.Setenv("WK_GITHUB_TOKEN", "wk-token")
	t.Setenv("GH_TOKEN", "gh-token")
	t.Setenv("GITHUB_TOKEN", "github-token")
	if got := githubToken(); got != "wk-token" {
		t.Fatalf("expected WK_GITHUB_TOKEN to win, got %q", got)
	}

	t.Setenv("WK_GITHUB_TOKEN", "")
	if got := githubToken(); got != "gh-token" {
		t.Fatalf("expected GH_TOKEN to win, got %q", got)
	}

	t.Setenv("GH_TOKEN", "")
	if got := githubToken(); got != "github-token" {
		t.Fatalf("expected GITHUB_TOKEN to win, got %q", got)
	}
}

func TestNormalizeTag(t *testing.T) {
	t.Parallel()

	if got := normalizeTag(""); got != "" {
		t.Fatalf("normalizeTag empty = %q", got)
	}
	if got := normalizeTag("1.2.3"); got != "v1.2.3" {
		t.Fatalf("normalizeTag no-prefix = %q", got)
	}
	if got := normalizeTag("v1.2.3"); got != "v1.2.3" {
		t.Fatalf("normalizeTag existing-prefix = %q", got)
	}
}

func TestDownloadFile(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer server.Close()

	dst := filepath.Join(t.TempDir(), "download.bin")
	if err := downloadFile(context.Background(), server.URL, dst); err != nil {
		t.Fatalf("downloadFile: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != "payload" {
		t.Fatalf("downloaded content mismatch: %q", string(got))
	}
}

func TestDownloadFile_HTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusBadRequest)
	}))
	defer server.Close()

	err := downloadFile(context.Background(), server.URL, filepath.Join(t.TempDir(), "download.bin"))
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "download failed (400)") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractBinaries_TarGz(t *testing.T) {
	t.Parallel()

	wkName, gogName := binaryNamesForTest()
	archivePath := filepath.Join(t.TempDir(), "bundle.tar.gz")
	if err := writeTarGzForTest(archivePath, map[string]string{
		wkName:      "wk-bytes",
		gogName:     "gog-bytes",
		"README.md": "ignore",
	}); err != nil {
		t.Fatalf("writeTarGzForTest: %v", err)
	}

	binaries, err := extractBinaries(archivePath)
	if err != nil {
		t.Fatalf("extractBinaries tar.gz: %v", err)
	}
	if string(binaries[wkName]) != "wk-bytes" {
		t.Fatalf("wk bytes mismatch: %q", string(binaries[wkName]))
	}
	if string(binaries[gogName]) != "gog-bytes" {
		t.Fatalf("gog bytes mismatch: %q", string(binaries[gogName]))
	}
}

func TestExtractBinaries_Zip(t *testing.T) {
	t.Parallel()

	wkName, gogName := binaryNamesForTest()
	archivePath := filepath.Join(t.TempDir(), "bundle.zip")
	if err := writeZipForTest(archivePath, map[string]string{
		wkName:      "wk-zip",
		gogName:     "gog-zip",
		"notes.txt": "ignore",
	}); err != nil {
		t.Fatalf("writeZipForTest: %v", err)
	}

	binaries, err := extractBinaries(archivePath)
	if err != nil {
		t.Fatalf("extractBinaries zip: %v", err)
	}
	if string(binaries[wkName]) != "wk-zip" {
		t.Fatalf("wk bytes mismatch: %q", string(binaries[wkName]))
	}
	if string(binaries[gogName]) != "gog-zip" {
		t.Fatalf("gog bytes mismatch: %q", string(binaries[gogName]))
	}
}

func TestExtractBinaries_MissingWK(t *testing.T) {
	t.Parallel()

	archivePath := filepath.Join(t.TempDir(), "bundle.tar.gz")
	if err := writeTarGzForTest(archivePath, map[string]string{
		"README.md": "only docs",
	}); err != nil {
		t.Fatalf("writeTarGzForTest: %v", err)
	}

	_, err := extractBinaries(archivePath)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "archive did not contain wk binary") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallBinaries(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wkName, gogName := binaryNamesForTest()
	selfPath := filepath.Join(dir, gogName)
	if err := os.WriteFile(selfPath, []byte("old"), 0o755); err != nil {
		t.Fatalf("write old alias: %v", err)
	}

	binaries := map[string][]byte{
		wkName:  []byte("new-wk"),
		gogName: []byte("new-gog"),
	}
	wkPath, err := installBinaries(binaries, selfPath)
	if err != nil {
		t.Fatalf("installBinaries: %v", err)
	}

	if wkPath != filepath.Join(dir, wkName) {
		t.Fatalf("wkPath mismatch: got %q", wkPath)
	}
	if got, err := os.ReadFile(filepath.Join(dir, wkName)); err != nil || string(got) != "new-wk" {
		t.Fatalf("wk content mismatch: err=%v got=%q", err, string(got))
	}
	if got, err := os.ReadFile(filepath.Join(dir, gogName)); err != nil || string(got) != "new-gog" {
		t.Fatalf("gog content mismatch: err=%v got=%q", err, string(got))
	}
}

func TestInstallBinaries_MissingWK(t *testing.T) {
	t.Parallel()

	_, gogName := binaryNamesForTest()
	_, err := installBinaries(map[string][]byte{
		gogName: []byte("gog-only"),
	}, filepath.Join(t.TempDir(), gogName))
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "missing wk binary") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCopyDir(t *testing.T) {
	t.Parallel()

	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested", "file.txt"), []byte("copy-me"), 0o640); err != nil {
		t.Fatalf("write src file: %v", err)
	}

	dst := filepath.Join(t.TempDir(), "dst")
	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "nested", "file.txt"))
	if err != nil {
		t.Fatalf("read copied file: %v", err)
	}
	if string(got) != "copy-me" {
		t.Fatalf("copied content mismatch: %q", string(got))
	}
}

func TestCopyDir_RejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows")
	}

	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "target.txt"), []byte("target"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink("target.txt", filepath.Join(src, "link.txt")); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	err := copyDir(src, filepath.Join(t.TempDir(), "dst"))
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported symlink") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateSkills(t *testing.T) {
	gitBinDir := t.TempDir()
	gitScript := filepath.Join(gitBinDir, "git")
	if err := os.WriteFile(gitScript, []byte(`#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "clone" ]]; then
  dst="${@: -1}"
  mkdir -p "$dst"
  if [[ "${WK_FAKE_GIT_WITH_SKILLS:-1}" == "1" ]]; then
    mkdir -p "$dst/skills/workit"
    printf 'router: workit\n' > "$dst/skills/workit/SKILL.md"
  fi
  if [[ -n "${WK_FAKE_GIT_ARGS_FILE:-}" ]]; then
    printf '%s\n' "$*" > "$WK_FAKE_GIT_ARGS_FILE"
  fi
  exit 0
fi
exit 0
`), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}

	t.Setenv("PATH", gitBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	argsFile := filepath.Join(t.TempDir(), "git.args")
	t.Setenv("WK_FAKE_GIT_ARGS_FILE", argsFile)
	t.Setenv("WK_FAKE_GIT_WITH_SKILLS", "1")

	dst := filepath.Join(t.TempDir(), "skills")
	state, err := updateSkills(context.Background(), "automagik-dev/workit", dst, "v1.2.3")
	if err != nil {
		t.Fatalf("updateSkills first run: %v", err)
	}
	if state != "installed" {
		t.Fatalf("expected installed, got %q", state)
	}
	if _, statErr := os.Stat(filepath.Join(dst, "workit", "SKILL.md")); statErr != nil {
		t.Fatalf("missing SKILL.md: %v", statErr)
	}
	argsRaw, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args file: %v", err)
	}
	if !strings.Contains(string(argsRaw), "--branch v1.2.3") {
		t.Fatalf("expected clone args to include branch: %q", string(argsRaw))
	}

	if writeErr := os.WriteFile(filepath.Join(dst, "workit", "STALE.txt"), []byte("stale"), 0o644); writeErr != nil {
		t.Fatalf("write stale marker: %v", writeErr)
	}
	state, err = updateSkills(context.Background(), "automagik-dev/workit", dst, "")
	if err != nil {
		t.Fatalf("updateSkills second run: %v", err)
	}
	if state != "updated" {
		t.Fatalf("expected updated, got %q", state)
	}
	if _, err := os.Stat(filepath.Join(dst, "workit", "STALE.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected stale marker to be removed, err=%v", err)
	}
}

func TestUpdateSkills_MissingGit(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := updateSkills(context.Background(), "automagik-dev/workit", filepath.Join(t.TempDir(), "skills"), "")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "git is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateSkills_MissingSkillFolder(t *testing.T) {
	gitBinDir := t.TempDir()
	gitScript := filepath.Join(gitBinDir, "git")
	if err := os.WriteFile(gitScript, []byte(`#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "clone" ]]; then
  dst="${@: -1}"
  mkdir -p "$dst"
  exit 0
fi
exit 0
`), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}

	t.Setenv("PATH", gitBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	_, err := updateSkills(context.Background(), "automagik-dev/workit", filepath.Join(t.TempDir(), "skills"), "")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "does not contain workit skill folder") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateCmdRun_SkipBinarySkillsOnly(t *testing.T) {
	gitBinDir := t.TempDir()
	gitScript := filepath.Join(gitBinDir, "git")
	if err := os.WriteFile(gitScript, []byte(`#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "clone" ]]; then
  dst="${@: -1}"
  mkdir -p "$dst/skills/workit"
  printf 'router: workit\n' > "$dst/skills/workit/SKILL.md"
  exit 0
fi
exit 0
`), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}

	t.Setenv("PATH", gitBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	dst := filepath.Join(t.TempDir(), "skills")
	cmd := &UpdateCmd{
		Tag:        "1.2.3",
		SkipBinary: true,
		SkillsRepo: "automagik-dev/workit",
		SkillsDir:  dst,
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "workit", "SKILL.md")); err != nil {
		t.Fatalf("expected skills install, got err=%v", err)
	}
}

func binaryNamesForTest() (wkName, gogName string) {
	if runtime.GOOS == "windows" {
		return "wk.exe", "gog.exe"
	}
	return "wk", "gog"
}

func writeTarGzForTest(path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			return err
		}
	}
	return nil
}

func writeZipForTest(path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(content)); err != nil {
			return err
		}
	}
	return nil
}
