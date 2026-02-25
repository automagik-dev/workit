package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/automagik-dev/workit/internal/config"
	"github.com/automagik-dev/workit/internal/outfmt"
	"github.com/automagik-dev/workit/internal/ui"
)

const (
	defaultUpdateRepo  = "automagik-dev/workit"
	defaultSkillsRepo  = defaultUpdateRepo
	githubAPIBase      = "https://api.github.com"
	githubReadmeHeader = "application/vnd.github+json"
	windowsOS          = "windows"
	statusUpdated      = "updated"
	statusSkipped      = "skipped"
	statusInstalled    = "installed"
)

// UpdateCmd updates the local wk binary and optional skills repository.
type UpdateCmd struct {
	Tag        string `help:"Release tag to install (default: latest release)"`
	Repo       string `help:"GitHub repository (owner/name) for release updates"`
	SkillsRepo string `name:"skills-repo" help:"GitHub repository (owner/name) for skills updates"`
	SkillsDir  string `name:"skills-dir" help:"Directory for local skills checkout"`
	SkipBinary bool   `name:"skip-binary" help:"Skip binary self-update"`
	SkipSkills bool   `name:"skip-skills" help:"Skip skills update"`
}

type updateResult struct {
	Repo         string `json:"repo"`
	RequestedTag string `json:"requested_tag,omitempty"`
	Tag          string `json:"tag,omitempty"`
	Asset        string `json:"asset,omitempty"`
	BinaryPath   string `json:"binary_path,omitempty"`
	Binary       string `json:"binary,omitempty"` // updated | skipped
	SkillsRepo   string `json:"skills_repo,omitempty"`
	SkillsDir    string `json:"skills_dir,omitempty"`
	Skills       string `json:"skills,omitempty"` // cloned | updated | skipped
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func (c *UpdateCmd) Run(ctx context.Context) error {
	if c.SkipBinary && c.SkipSkills {
		return usage("nothing to do: both --skip-binary and --skip-skills are set")
	}

	repo := strings.TrimSpace(c.Repo)
	if repo == "" {
		repo = strings.TrimSpace(os.Getenv("WK_UPDATE_REPO"))
	}
	if repo == "" {
		repo = defaultUpdateRepo
	}
	skillsRepo := strings.TrimSpace(c.SkillsRepo)
	if skillsRepo == "" {
		skillsRepo = strings.TrimSpace(os.Getenv("WK_SKILLS_REPO"))
	}
	if skillsRepo == "" {
		skillsRepo = defaultSkillsRepo
	}

	skillsDir := strings.TrimSpace(c.SkillsDir)
	if skillsDir == "" {
		skillsDir = strings.TrimSpace(os.Getenv("WK_SKILLS_DIR"))
	}
	if skillsDir == "" {
		configDir, err := config.Dir()
		if err != nil {
			return fmt.Errorf("resolve config dir: %w", err)
		}
		skillsDir = filepath.Join(configDir, "skills")
	}

	result := updateResult{
		Repo:         repo,
		RequestedTag: strings.TrimSpace(c.Tag),
		SkillsRepo:   skillsRepo,
		SkillsDir:    skillsDir,
		Binary:       statusSkipped,
		Skills:       statusSkipped,
	}

	if !c.SkipBinary {
		tag, assetName, binaryPath, err := updateBinary(ctx, repo, strings.TrimSpace(c.Tag))
		if err != nil {
			return err
		}
		result.Tag = tag
		result.Asset = assetName
		result.BinaryPath = binaryPath
		result.Binary = statusUpdated
	}

	if !c.SkipSkills {
		skillsRef := result.Tag
		if strings.TrimSpace(skillsRef) == "" {
			skillsRef = normalizeTag(strings.TrimSpace(c.Tag))
		}

		state, err := updateSkills(ctx, skillsRepo, skillsDir, skillsRef)
		if err != nil {
			return err
		}
		result.Skills = state
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, result)
	}

	if u := ui.FromContext(ctx); u != nil {
		if result.Binary == statusUpdated {
			u.Out().Printf("binary updated: %s (%s)", result.Tag, result.Asset)
			u.Out().Printf("installed: %s", result.BinaryPath)
		}
		if result.Skills != statusSkipped {
			u.Out().Printf("skills %s: %s -> %s", result.Skills, result.SkillsRepo, result.SkillsDir)
		}
		return nil
	}

	if result.Binary == statusUpdated {
		fmt.Fprintf(os.Stdout, "binary updated: %s (%s)\n", result.Tag, result.Asset)
		fmt.Fprintf(os.Stdout, "installed: %s\n", result.BinaryPath)
	}
	if result.Skills != statusSkipped {
		fmt.Fprintf(os.Stdout, "skills %s: %s -> %s\n", result.Skills, result.SkillsRepo, result.SkillsDir)
	}

	return nil
}

func updateBinary(ctx context.Context, repo, version string) (tag string, assetName string, binaryPath string, err error) {
	if runtime.GOOS == windowsOS {
		return "", "", "", errors.New("wk update binary self-replacement is not supported on windows yet")
	}

	release, err := fetchRelease(ctx, repo, version)
	if err != nil {
		return "", "", "", err
	}
	if strings.TrimSpace(release.TagName) == "" {
		return "", "", "", errors.New("release metadata missing tag_name")
	}

	assetName, err = releaseAssetName(release.TagName, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", "", "", err
	}
	asset, ok := findAssetByName(release, assetName)
	if !ok {
		return "", "", "", fmt.Errorf("release %s has no asset %q", release.TagName, assetName)
	}

	tmpDir, err := os.MkdirTemp("", "wk-update-*")
	if err != nil {
		return "", "", "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, asset.Name)
	if dlErr := downloadFile(ctx, asset.BrowserDownloadURL, archivePath); dlErr != nil {
		return "", "", "", dlErr
	}

	binaries, err := extractBinaries(archivePath)
	if err != nil {
		return "", "", "", err
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", "", "", fmt.Errorf("resolve executable path: %w", err)
	}
	if resolved, evalErr := filepath.EvalSymlinks(exePath); evalErr == nil && resolved != "" {
		exePath = resolved
	}

	binaryPath, err = installBinaries(binaries, exePath)
	if err != nil {
		return "", "", "", err
	}

	return release.TagName, assetName, binaryPath, nil
}

func updateSkills(ctx context.Context, repo, dst, ref string) (string, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return "", errors.New("git is required for skills update (install git or use --skip-skills)")
	}

	tmpDir, err := os.MkdirTemp("", "wk-skills-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir for skills: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := filepath.Join(tmpDir, "repo")
	cloneURL := fmt.Sprintf("https://github.com/%s.git", repo)
	cloneArgs := []string{"clone", "--depth=1"}
	if strings.TrimSpace(ref) != "" {
		cloneArgs = append(cloneArgs, "--branch", ref)
	}
	cloneArgs = append(cloneArgs, cloneURL, repoDir)

	out, cmdErr := exec.CommandContext(ctx, gitPath, cloneArgs...).CombinedOutput() //nolint:gosec // args fixed by command implementation
	if cmdErr != nil {
		return "", fmt.Errorf("clone skills repository %s: %w\n%s", cloneURL, cmdErr, strings.TrimSpace(string(out)))
	}

	src := ""
	candidates := []string{
		filepath.Join(repoDir, "skills", "workit"),
		filepath.Join(repoDir, "workit"),
	}
	for _, candidate := range candidates {
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			src = candidate
			break
		}
	}
	if src == "" {
		return "", fmt.Errorf("skills repository %s does not contain workit skill folder", repo)
	}

	if err := os.MkdirAll(dst, 0o700); err != nil {
		return "", fmt.Errorf("ensure skills directory: %w", err)
	}

	target := filepath.Join(dst, "workit")
	_, existedErr := os.Stat(target)
	existed := existedErr == nil
	if existedErr != nil && !os.IsNotExist(existedErr) {
		return "", fmt.Errorf("check existing skill dir: %w", existedErr)
	}

	if err := os.RemoveAll(target); err != nil {
		return "", fmt.Errorf("clear existing skill dir: %w", err)
	}
	if err := copyDir(src, target); err != nil {
		return "", err
	}

	if existed {
		return statusUpdated, nil
	}
	return statusInstalled, nil
}

func fetchRelease(ctx context.Context, repo, version string) (githubRelease, error) {
	path := fmt.Sprintf("/repos/%s/releases/latest", repo)
	if version != "" {
		tag := strings.TrimSpace(version)
		if !strings.HasPrefix(tag, "v") {
			tag = "v" + tag
		}
		path = fmt.Sprintf("/repos/%s/releases/tags/%s", repo, tag)
	}

	url := githubAPIBase + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return githubRelease{}, fmt.Errorf("create release request: %w", err)
	}
	req.Header.Set("Accept", githubReadmeHeader)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "wk-update")

	if token := githubToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return githubRelease{}, fmt.Errorf("request release metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return githubRelease{}, fmt.Errorf("github release request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubRelease{}, fmt.Errorf("decode release metadata: %w", err)
	}

	return release, nil
}

func githubToken() string {
	for _, key := range []string{"WK_GITHUB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func findAssetByName(release githubRelease, name string) (githubAsset, bool) {
	for _, asset := range release.Assets {
		if strings.EqualFold(strings.TrimSpace(asset.Name), strings.TrimSpace(name)) {
			return asset, true
		}
	}
	return githubAsset{}, false
}

func normalizeTag(raw string) string {
	tag := strings.TrimSpace(raw)
	if tag == "" {
		return ""
	}
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return tag
}

func releaseAssetName(tag, goos, goarch string) (string, error) {
	releaseVersion := strings.TrimPrefix(strings.TrimSpace(tag), "v")
	if releaseVersion == "" {
		return "", errors.New("release tag is empty")
	}

	arch := strings.TrimSpace(goarch)
	switch arch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("unsupported architecture %q", goarch)
	}

	var ext string
	switch strings.TrimSpace(goos) {
	case "linux", "darwin":
		ext = "tar.gz"
	case windowsOS:
		ext = "zip"
	default:
		return "", fmt.Errorf("unsupported OS %q", goos)
	}

	return fmt.Sprintf("workit_%s_%s_%s.%s", releaseVersion, goos, arch, ext), nil
}

func downloadFile(ctx context.Context, url, dst string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}
	req.Header.Set("User-Agent", "wk-update")
	if token := githubToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("download failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	//nolint:gosec // destination path is controlled by the caller of update flow.
	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create archive file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write archive file: %w", err)
	}

	return nil
}

func extractBinaries(archivePath string) (map[string][]byte, error) {
	wanted := map[string]bool{}
	for _, name := range wantedBinaryNames() {
		wanted[name] = true
	}
	out := map[string][]byte{}

	switch {
	case strings.HasSuffix(archivePath, ".tar.gz"):
		//nolint:gosec // archivePath is created in a temp directory by this command.
		f, err := os.Open(archivePath)
		if err != nil {
			return nil, fmt.Errorf("open archive: %w", err)
		}
		defer f.Close()

		gzr, err := gzip.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("open gzip archive: %w", err)
		}
		defer gzr.Close()

		tr := tar.NewReader(gzr)
		for {
			h, err := tr.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("read tar archive: %w", err)
			}
			if h == nil || h.FileInfo().IsDir() {
				continue
			}

			base := filepath.Base(h.Name)
			if !wanted[base] {
				continue
			}
			b, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("read %s from archive: %w", base, err)
			}
			out[base] = b
		}

	case strings.HasSuffix(archivePath, ".zip"):
		zr, err := zip.OpenReader(archivePath)
		if err != nil {
			return nil, fmt.Errorf("open zip archive: %w", err)
		}
		defer zr.Close()

		for _, f := range zr.File {
			base := filepath.Base(f.Name)
			if !wanted[base] {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("open %s in zip archive: %w", base, err)
			}
			b, readErr := io.ReadAll(rc)
			closeErr := rc.Close()
			if readErr != nil {
				return nil, fmt.Errorf("read %s from zip archive: %w", base, readErr)
			}
			if closeErr != nil {
				return nil, fmt.Errorf("close %s from zip archive: %w", base, closeErr)
			}
			out[base] = b
		}

	default:
		return nil, fmt.Errorf("unsupported archive format: %s", archivePath)
	}

	if len(out) == 0 {
		return nil, errors.New("archive did not contain wk binary")
	}
	return out, nil
}

func wantedBinaryNames() []string {
	if runtime.GOOS == windowsOS {
		return []string{"wk.exe", "gog.exe"}
	}
	return []string{"wk", "gog"}
}

func installBinaries(binaries map[string][]byte, selfPath string) (string, error) {
	dir := filepath.Dir(selfPath)
	wkName := "wk"
	if runtime.GOOS == windowsOS {
		wkName = "wk.exe"
	}
	wkPath := filepath.Join(dir, wkName)

	wkBytes, ok := binaries[wkName]
	if !ok {
		return "", errors.New("downloaded archive is missing wk binary")
	}
	if err := writeAtomicExecutable(wkPath, wkBytes); err != nil {
		return "", err
	}

	gogName := "gog"
	if runtime.GOOS == windowsOS {
		gogName = "gog.exe"
	}
	if gogBytes, ok := binaries[gogName]; ok {
		_ = writeAtomicExecutable(filepath.Join(dir, gogName), gogBytes)
	}

	// If command is being run as an alias path distinct from wk, refresh that
	// path too when possible.
	base := filepath.Base(selfPath)
	if base != wkName {
		if aliasBytes, ok := binaries[base]; ok {
			_ = writeAtomicExecutable(selfPath, aliasBytes)
		} else {
			_ = writeAtomicExecutable(selfPath, wkBytes)
		}
	}

	return wkPath, nil
}

func writeAtomicExecutable(path string, content []byte) error {
	tmp := path + ".tmp-" + fmt.Sprintf("%d", os.Getpid())
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("write %s: %w (try running with permissions that can write this path)", path, err)
		}
		return fmt.Errorf("write temp executable %s: %w", tmp, err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		if os.IsPermission(err) {
			return fmt.Errorf("replace %s: %w (try running with permissions that can write this path)", path, err)
		}
		return fmt.Errorf("replace executable %s: %w", path, err)
	}
	//nolint:gosec // installed executable must remain executable for users.
	if err := os.Chmod(path, 0o755); err != nil {
		return fmt.Errorf("set executable mode on %s: %w", path, err)
	}

	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return fmt.Errorf("compute relative path: %w", relErr)
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			if mkErr := os.MkdirAll(target, 0o750); mkErr != nil {
				return fmt.Errorf("create directory %s: %w", target, mkErr)
			}
			return nil
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsupported symlink in skills repo: %s", path)
		}

		if mkErr := os.MkdirAll(filepath.Dir(target), 0o750); mkErr != nil {
			return fmt.Errorf("create parent directory %s: %w", filepath.Dir(target), mkErr)
		}
		//nolint:gosec // source path comes from a cloned local repository.
		in, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open source file %s: %w", path, err)
		}

		//nolint:gosec // target path is derived from controlled repo content.
		out, err := os.Create(target)
		if err != nil {
			_ = in.Close()
			return fmt.Errorf("create destination file %s: %w", target, err)
		}

		if _, err := io.Copy(out, in); err != nil {
			_ = in.Close()
			_ = out.Close()
			return fmt.Errorf("copy %s to %s: %w", path, target, err)
		}
		if err := in.Close(); err != nil {
			_ = out.Close()
			return fmt.Errorf("close source file %s: %w", path, err)
		}
		if err := out.Close(); err != nil {
			return fmt.Errorf("close destination file %s: %w", target, err)
		}
		if err := os.Chmod(target, info.Mode().Perm()); err != nil {
			return fmt.Errorf("set file mode on %s: %w", target, err)
		}

		return nil
	})
}
