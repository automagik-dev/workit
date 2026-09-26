// Package releasescripts_test drives the release shell scripts in
// .github/scripts against throwaway git repositories and a fake `gh`, so the
// tagging and "Latest" rules are checked by `make ci` instead of only by a
// live release.
package releasescripts_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const (
	nextTagScript   = "../../.github/scripts/next-release-tag.sh"
	setLatestScript = "../../.github/scripts/set-latest-release.sh"
	fakeRepo        = "automagik-dev/workit"
)

func requireTools(t *testing.T, tools ...string) {
	t.Helper()

	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not found in PATH: %v", tool, err)
		}
	}
}

// gitEnv isolates git from the developer's config and identity.
func gitEnv(t *testing.T) []string {
	t.Helper()

	home := t.TempDir()

	return append(os.Environ(),
		"HOME="+home,
		"GIT_CONFIG_GLOBAL="+filepath.Join(home, "gitconfig"),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_TERMINAL_PROMPT=0",
	)
}

type repo struct {
	t   *testing.T
	dir string
	env []string
}

func newRepo(t *testing.T) *repo {
	t.Helper()

	r := &repo{t: t, dir: t.TempDir(), env: gitEnv(t)}
	r.git("init", "-q", "-b", "main")
	r.commit("base")

	return r
}

func (r *repo) git(args ...string) {
	r.t.Helper()

	cmd := exec.CommandContext(r.t.Context(), "git", args...)
	cmd.Dir = r.dir
	cmd.Env = r.env

	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func (r *repo) commit(message string) {
	r.t.Helper()
	r.git("commit", "-q", "--allow-empty", "-m", message)
}

// merge adds a feature branch with one commit and merges it into main with
// a merge commit carrying message, the way a PR merge lands.
func (r *repo) merge(branch string, featureMessage string, message string) {
	r.t.Helper()
	r.git("checkout", "-q", "-b", branch)
	r.commit(featureMessage)
	r.git("checkout", "-q", "main")
	r.git("merge", "-q", "--no-ff", branch, "-m", message)
}

func (r *repo) tag(name string) {
	r.t.Helper()
	r.git("tag", "-a", name, "-m", "Release "+name)
}

type scriptResult struct {
	rc  int
	out string
}

func runScript(t *testing.T, dir string, env []string, script string) scriptResult {
	t.Helper()

	abs, err := filepath.Abs(script)
	if err != nil {
		t.Fatalf("abs %s: %v", script, err)
	}

	cmd := exec.CommandContext(t.Context(), "bash", abs)
	cmd.Dir = dir
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err == nil {
		return scriptResult{rc: 0, out: string(out)}
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("run %s: %v\n%s", script, err, out)
	}

	return scriptResult{rc: exitErr.ExitCode(), out: string(out)}
}

// ---- next-release-tag.sh ----

func nextTag(t *testing.T, r *repo) (string, string) {
	t.Helper()

	res := runScript(t, r.dir, append(r.env, "RELEASE_DATE=260926"), nextTagScript)
	if res.rc != 0 {
		t.Fatalf("next-release-tag.sh rc=%d\n%s", res.rc, res.out)
	}

	for _, line := range strings.Split(res.out, "\n") {
		if tag, ok := strings.CutPrefix(line, "tag="); ok {
			return tag, res.out
		}
	}

	t.Fatalf("no tag= line in output:\n%s", res.out)

	return "", ""
}

func TestNextReleaseTag(t *testing.T) {
	requireTools(t, "bash", "git")
	t.Parallel()

	tests := []struct {
		name  string
		setup func(r *repo)
		want  string
	}{
		{
			name: "no tags yet",
			setup: func(r *repo) {
				r.merge("a", "feat a", "Merge pull request #1\n\nfeat: a")
			},
			want: "v2.260926.1",
		},
		{
			name: "next number today",
			setup: func(r *repo) {
				r.tag("v2.260926.1")
				r.merge("a", "feat a", "Merge pull request #1\n\nfeat: a")
				r.tag("v2.260926.2")
				r.merge("b", "feat b", "Merge pull request #2\n\nfeat: b")
			},
			want: "v2.260926.3",
		},
		{
			name: "first release of a new day",
			setup: func(r *repo) {
				r.tag("v2.260925.7")
				r.merge("a", "feat a", "Merge pull request #1\n\nfeat: a")
			},
			want: "v2.260926.1",
		},
		{
			name: "HEAD already tagged",
			setup: func(r *repo) {
				r.merge("a", "feat a", "Merge pull request #1\n\nfeat: a")
				r.tag("v2.260926.1")
			},
			want: "",
		},
		{
			// Re-running an older run on A after B was tagged: B's tag
			// already contains A, so A must not get a new tag.
			name: "re-run on a commit a newer tag contains",
			setup: func(r *repo) {
				r.tag("v2.260926.1")
				r.merge("a", "feat a", "Merge pull request #1\n\nfeat: a")
				r.merge("b", "feat b", "Merge pull request #2\n\nfeat: b")
				r.tag("v2.260926.2")
				r.git("checkout", "-q", "--detach", "HEAD~1")
			},
			want: "",
		},
		{
			// The review case: B's pending run was replaced by C's, and C asks
			// to skip. B still wants a release, so HEAD (C) is tagged.
			name: "skip head replaces a pending release",
			setup: func(r *repo) {
				r.tag("v2.260926.1")
				r.merge("b", "feat b", "Merge pull request #2\n\nfeat: b")
				r.merge("c", "docs c", "Merge pull request #3\n\ndocs: c [skip release]")
			},
			want: "v2.260926.2",
		},
		{
			name: "every untagged commit asks to skip",
			setup: func(r *repo) {
				r.tag("v2.260926.1")
				r.merge("b", "docs b", "Merge pull request #2\n\ndocs: b [skip release]")
				r.merge("c", "docs c", "Merge pull request #3\n\nchore: c [skip ci]")
			},
			want: "",
		},
		{
			// The skip marker is read from first-parent commits only (one per
			// push or merge), like head_commit.message was.
			name: "skip marker on a PR branch commit does not skip the merge",
			setup: func(r *repo) {
				r.tag("v2.260926.1")
				r.merge("b", "wip [skip ci]", "Merge pull request #2\n\nfeat: b")
			},
			want: "v2.260926.2",
		},
		{
			name: "skip marker on the merge skips its branch commits",
			setup: func(r *repo) {
				r.tag("v2.260926.1")
				r.merge("b", "feat b", "Merge pull request #2\n\ndocs: b [skip release]")
			},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := newRepo(t)
			tc.setup(r)

			got, out := nextTag(t, r)
			if got != tc.want {
				t.Fatalf("tag = %q, want %q\n%s", got, tc.want, out)
			}
		})
	}
}

// ---- set-latest-release.sh ----

// fakeGH answers the three gh calls set-latest-release.sh makes from files
// in its directory: releases.json (the release listing), latest (the current
// Latest tag, empty for none), list-error and latest-error (API failures).
// `release edit --latest` rewrites latest and appends to edits.
const fakeGH = `#!/usr/bin/env bash
set -euo pipefail
d="$FAKE_GH_DIR"
if [ "$1" = api ]; then
  shift
  path=""
  filter=""
  while [ $# -gt 0 ]; do
    case "$1" in
      --paginate) shift ;;
      --jq) filter="$2"; shift 2 ;;
      *) path="$1"; shift ;;
    esac
  done
  case "$path" in
    "repos/` + fakeRepo + `/releases?per_page=100")
      if [ -f "$d/list-error" ]; then echo "gh: Bad Gateway (HTTP 502)" >&2; exit 1; fi
      jq -r "$filter" "$d/releases.json" ;;
    "repos/` + fakeRepo + `/releases/latest")
      if [ -f "$d/latest-error" ]; then echo "error connecting to api.github.com" >&2; exit 1; fi
      latest="$(cat "$d/latest")"
      if [ -z "$latest" ]; then
        echo '{"message":"Not Found","status":"404"}'
        echo "gh: Not Found (HTTP 404)" >&2
        exit 1
      fi
      jq -rn --arg t "$latest" '{tag_name: $t}' | jq -r "$filter" ;;
    *) echo "fake gh: unexpected api path $path" >&2; exit 2 ;;
  esac
elif [ "$1 $2 $4 $6" = "release edit -R --latest" ] && [ "$5" = "` + fakeRepo + `" ]; then
  printf '%s' "$3" > "$d/latest"
  echo "$3" >> "$d/edits"
else
  echo "fake gh: unexpected call: $*" >&2
  exit 2
fi
`

type latestCase struct {
	name     string
	setup    func(r *repo) // extra history after v2.260926.3/.4/.5
	releases map[string]string
	latest   string
	own      string
	listErr  bool
	lateErr  bool

	wantRC     int
	wantLatest string
	wantEdits  []string
	wantOut    []string
}

type fakeRelease struct {
	TagName    string           `json:"tag_name"`
	Draft      bool             `json:"draft"`
	Prerelease bool             `json:"prerelease"`
	Assets     []map[string]any `json:"assets"`
}

func writeReleases(t *testing.T, path string, states map[string]string) {
	t.Helper()

	releases := make([]fakeRelease, 0, len(states))

	for tag, state := range states {
		rel := fakeRelease{TagName: tag, Assets: []map[string]any{{"name": "checksums.txt"}}}

		switch state {
		case "published":
		case "draft":
			rel.Draft = true
		case "prerelease":
			rel.Prerelease = true
		case "without-assets":
			rel.Assets = []map[string]any{}
		default:
			t.Fatalf("unknown release state %q", state)
		}

		releases = append(releases, rel)
	}

	data, err := json.Marshal(releases)
	if err != nil {
		t.Fatalf("marshal releases: %v", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write releases: %v", err)
	}
}

func runSetLatest(t *testing.T, tc latestCase) {
	t.Helper()

	// main: base, then merges #3, #4, #5 tagged v2.260926.3/.4/.5.
	work := newRepo(t)
	for _, n := range []int{3, 4, 5} {
		work.merge(fmt.Sprintf("f%d", n), fmt.Sprintf("feat %d", n), fmt.Sprintf("Merge pull request #%d", n))
		work.tag(fmt.Sprintf("v2.260926.%d", n))
	}

	if tc.setup != nil {
		tc.setup(work)
	}

	origin := t.TempDir()
	work.git("init", "-q", "--bare", origin)
	work.git("push", "-q", origin, "main", "--tags")

	runner := &repo{t: t, dir: t.TempDir(), env: work.env}
	runner.git("clone", "-q", origin, ".")

	fake := t.TempDir()
	if err := os.WriteFile(filepath.Join(fake, "gh"), []byte(fakeGH), 0o700); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}

	writeReleases(t, filepath.Join(fake, "releases.json"), tc.releases)

	if err := os.WriteFile(filepath.Join(fake, "latest"), []byte(tc.latest), 0o600); err != nil {
		t.Fatalf("write latest: %v", err)
	}

	for flag, on := range map[string]bool{"list-error": tc.listErr, "latest-error": tc.lateErr} {
		if on {
			if err := os.WriteFile(filepath.Join(fake, flag), nil, 0o600); err != nil {
				t.Fatalf("write %s: %v", flag, err)
			}
		}
	}

	env := slices.Concat(runner.env, []string{
		"PATH=" + fake + string(os.PathListSeparator) + os.Getenv("PATH"),
		"FAKE_GH_DIR=" + fake,
		"GITHUB_REPOSITORY=" + fakeRepo,
		"RELEASE_TAG=" + tc.own,
	})

	res := runScript(t, runner.dir, env, setLatestScript)

	if res.rc != tc.wantRC {
		t.Fatalf("rc = %d, want %d\n%s", res.rc, tc.wantRC, res.out)
	}

	latest, err := os.ReadFile(filepath.Join(fake, "latest"))
	if err != nil {
		t.Fatalf("read latest: %v", err)
	}

	if string(latest) != tc.wantLatest {
		t.Fatalf("Latest = %q, want %q\n%s", latest, tc.wantLatest, res.out)
	}

	var edits []string

	if data, err := os.ReadFile(filepath.Join(fake, "edits")); err == nil {
		edits = strings.Fields(string(data))
	}

	if !reflect.DeepEqual(edits, tc.wantEdits) {
		t.Fatalf("edits = %v, want %v\n%s", edits, tc.wantEdits, res.out)
	}

	for _, want := range tc.wantOut {
		if !strings.Contains(res.out, want) {
			t.Fatalf("output lacks %q:\n%s", want, res.out)
		}
	}
}

func TestSetLatestRelease(t *testing.T) {
	requireTools(t, "bash", "git", "jq", "awk")
	t.Parallel()

	all := map[string]string{"v2.260926.3": "published", "v2.260926.4": "published", "v2.260926.5": "published"}

	with := func(tag string, state string) map[string]string {
		m := map[string]string{"v2.260926.3": "published", "v2.260926.4": "published"}
		if state != "missing" {
			m[tag] = state
		}

		return m
	}

	tests := []latestCase{
		{
			// The incident: .4 finished last and took Latest.
			name: "incident: Latest on .4, .5 published", releases: all, latest: "v2.260926.4", own: "v2.260926.4",
			wantLatest: "v2.260926.5", wantEdits: []string{"v2.260926.5"},
		},
		{
			name: "already Latest", releases: all, latest: "v2.260926.5", own: "v2.260926.5",
			wantLatest: "v2.260926.5",
		},
		{
			name: "no Latest yet (404)", releases: all, latest: "", own: "v2.260926.5",
			wantLatest: "v2.260926.5", wantEdits: []string{"v2.260926.5"},
		},
		{
			// .4's run finishes while .5 is still building: fall back to .4.
			name: "newest release missing, other run", releases: with("v2.260926.5", "missing"), latest: "v2.260926.3", own: "v2.260926.4",
			wantLatest: "v2.260926.4", wantEdits: []string{"v2.260926.4"},
			wantOut: []string{"::warning::v2.260926.5", "its release is missing"},
		},
		{
			// .5's own run finished GoReleaser but its release is absent.
			name: "newest release missing, own run fails", releases: with("v2.260926.5", "missing"), latest: "v2.260926.3", own: "v2.260926.5",
			wantRC: 1, wantLatest: "v2.260926.4", wantEdits: []string{"v2.260926.4"},
			wantOut: []string{"::warning::v2.260926.5", "::error::This run's tag v2.260926.5"},
		},
		{
			// A GoReleaser run that fails mid-upload leaves a draft.
			name: "newest build failed (draft), own run fails", releases: with("v2.260926.5", "draft"), latest: "v2.260926.3", own: "v2.260926.5",
			wantRC: 1, wantLatest: "v2.260926.4", wantEdits: []string{"v2.260926.4"},
			wantOut: []string{"its release is draft", "::error::This run's tag v2.260926.5"},
		},
		{
			name: "newest draft, other run", releases: with("v2.260926.5", "draft"), latest: "v2.260926.4", own: "v2.260926.4",
			wantLatest: "v2.260926.4",
			wantOut:    []string{"::warning::v2.260926.5", "its release is draft", "Latest stays v2.260926.4"},
		},
		{
			name: "newest without assets", releases: with("v2.260926.5", "without-assets"), latest: "v2.260926.3", own: "v2.260926.4",
			wantLatest: "v2.260926.4", wantEdits: []string{"v2.260926.4"},
			wantOut: []string{"its release is without-assets"},
		},
		{
			name: "newest prerelease", releases: with("v2.260926.5", "prerelease"), latest: "v2.260926.3", own: "v2.260926.4",
			wantLatest: "v2.260926.4", wantEdits: []string{"v2.260926.4"},
			wantOut: []string{"its release is prerelease"},
		},
		{
			name: "release listing API error", releases: all, latest: "v2.260926.4", own: "v2.260926.5", listErr: true,
			wantRC: 1, wantLatest: "v2.260926.4",
			wantOut: []string{"::error::Could not list the releases"},
		},
		{
			name: "Latest lookup API error", releases: all, latest: "v2.260926.4", own: "v2.260926.5", lateErr: true,
			wantRC: 1, wantLatest: "v2.260926.4",
			wantOut: []string{"error connecting to api.github.com", "::error::Could not read the current Latest"},
		},
		{
			// Never backwards: .6 is missing and Latest is .5, so the fallback
			// stops at .5 instead of reaching .4.
			name: "never moves backwards past current Latest",
			setup: func(r *repo) {
				r.merge("f6", "feat 6", "Merge pull request #6")
				r.tag("v2.260926.6")
			},
			releases: all, latest: "v2.260926.5", own: "v2.260926.6",
			wantRC: 1, wantLatest: "v2.260926.5",
			wantOut: []string{"::warning::v2.260926.6", "Latest stays v2.260926.5"},
		},
		{
			name: "older re-release does not move Latest back", releases: all, latest: "v2.260926.5", own: "v2.260926.3",
			wantLatest: "v2.260926.5",
		},
		{
			name: "untagged commits past the newest tag",
			setup: func(r *repo) {
				r.commit("docs [skip release]")
			},
			releases: all, latest: "v2.260926.4", own: "v2.260926.5",
			wantLatest: "v2.260926.5", wantEdits: []string{"v2.260926.5"},
		},
		{
			name: "newer-numbered tag off main is ignored",
			setup: func(r *repo) {
				r.git("checkout", "-q", "-b", "side", "HEAD~1")
				r.commit("off main")
				r.tag("v2.260926.9")
				r.git("checkout", "-q", "main")
			},
			releases: map[string]string{"v2.260926.4": "published", "v2.260926.5": "published", "v2.260926.9": "published"},
			latest:   "v2.260926.4", own: "v2.260926.9",
			wantLatest: "v2.260926.5", wantEdits: []string{"v2.260926.5"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runSetLatest(t, tc)
		})
	}
}
