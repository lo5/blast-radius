package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/*.golden")

// testdata is absolute because tests change directory before comparing.
var testdata, _ = filepath.Abs(filepath.Join("..", "..", "testdata"))

// testEnv is the fixed environment for git and the CLI in e2e tests.
func testEnv(t *testing.T) []string {
	t.Helper()
	const date = "2026-01-01T00:00:00Z"
	return []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + t.TempDir(),
		"LC_ALL=C",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CEILING_DIRECTORIES=" + os.TempDir(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_AUTHOR_DATE=" + date,
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com", "GIT_COMMITTER_DATE=" + date,
		"COLUMNS=100",
	}
}

// newRepo creates an empty repository with branch main in a temp dir.
func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	return dir
}

// git runs git in dir under the test environment.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = testEnv(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// write creates or overwrites dir/path, making parent directories.
func write(t *testing.T, dir, path, content string) {
	t.Helper()
	p := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// lines returns n newline-terminated lines "<prefix> <i>".
func lines(prefix string, n int) string {
	var b strings.Builder
	for i := range n {
		fmt.Fprintf(&b, "%s %d\n", prefix, i)
	}
	return b.String()
}

// commitAll stages everything in dir and commits it.
func commitAll(t *testing.T, dir, msg string) {
	t.Helper()
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", msg)
}

// runCLI runs the CLI in-process from dir with --color=never prepended and
// returns stdout, plus stderr and the exit code when either is notable.
func runCLI(t *testing.T, dir string, args ...string) string {
	t.Helper()
	return runCLIEnv(t, dir, testEnv(t), args...)
}

// runCLIEnv is runCLI with the given environment.
func runCLIEnv(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	code := run(append([]string{"--color=never"}, args...), &stdout, &stderr, env)
	out := stdout.String()
	if stderr.Len() > 0 || code != 0 {
		out += fmt.Sprintf("-- stderr --\n%s-- exit %d --\n", stderr.String(), code)
	}
	return out
}

// golden compares got with testdata/<name>.golden, rewriting it with -update.
func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join(testdata, name+".golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run go test ./... -update)", err)
	}
	if got != string(want) {
		t.Errorf("output differs from %s (run go test ./... -update)\ngot:\n%s\nwant:\n%s", path, got, want)
	}
}

func TestUncommitted(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "README.md", lines("readme", 10))
	write(t, dir, "go.mod", "module example.com/x\n\ngo 1.27\n")
	write(t, dir, "src/api/handler.go", lines("handler", 50))
	write(t, dir, "src/core/core.go", lines("core", 40))
	write(t, dir, "docs/guide.md", lines("guide", 25))
	write(t, dir, "my dir/notes ü.txt", lines("note", 3))
	commitAll(t, dir, "initial")

	// Staged: a modification and a large new file.
	write(t, dir, "src/api/handler.go", lines("handler", 30)+lines("new handler", 20))
	write(t, dir, "src/api/routes.go", lines("route", 1500))
	// Staged, then edited again (unstaged).
	write(t, dir, "src/core/core.go", lines("core", 40)+lines("more", 5))
	git(t, dir, "add", "src")
	write(t, dir, "src/core/core.go", lines("core", 35)+lines("more", 5))
	// Unstaged: a modification and a deletion.
	write(t, dir, "README.md", lines("readme", 9)+lines("changed", 2))
	write(t, dir, "my dir/notes ü.txt", lines("note", 3)+"extra\n")
	if err := os.Remove(filepath.Join(dir, "docs/guide.md")); err != nil {
		t.Fatal(err)
	}
	// Untracked: counted as all added, except ignored ones.
	write(t, dir, ".gitignore", "*.log\n")
	write(t, dir, "src/api/new.go", lines("new", 30))
	write(t, dir, "scripts/run.sh", "#!/bin/sh\necho run")
	write(t, dir, "debug.log", lines("log", 100))

	golden(t, "uncommitted", runCLI(t, dir))
}

// A repo with no commits diffs staged files against the empty tree.
func TestNoCommits(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "src/main.go", lines("main", 20))
	write(t, dir, "README.md", lines("readme", 5))
	git(t, dir, "add", "src")
	golden(t, "no-commits", runCLI(t, dir))
}

// Binary files are dropped, tracked or untracked.
func TestBinaryIgnored(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "assets/logo.png", "\x89PNG\x00old")
	write(t, dir, "src/a.go", lines("a", 10))
	commitAll(t, dir, "initial")
	write(t, dir, "assets/logo.png", "\x89PNG\x00new\n")
	write(t, dir, "assets/icon.ico", "\x00\x00\x01\x00")
	write(t, dir, "src/a.go", lines("a", 12))
	golden(t, "binary-ignored", runCLI(t, dir))
}

// An untracked file that can't be read is skipped with a warning.
func TestUnreadableUntracked(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can read anything")
	}
	dir := newRepo(t)
	write(t, dir, "a.txt", "a\n")
	commitAll(t, dir, "initial")
	write(t, dir, "b.txt", lines("b", 3))
	write(t, dir, "secret.txt", lines("secret", 3))
	if err := os.Chmod(filepath.Join(dir, "secret.txt"), 0o000); err != nil {
		t.Fatal(err)
	}
	golden(t, "unreadable-untracked", runCLI(t, dir))
}

func TestNoChanges(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "a.txt", "a\n")
	commitAll(t, dir, "initial")
	golden(t, "no-changes", runCLI(t, dir))
}

func TestNotARepo(t *testing.T) {
	golden(t, "not-a-repo", runCLI(t, t.TempDir()))
}

func TestGitNotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var stdout, stderr bytes.Buffer
	code := run(nil, &stdout, &stderr, nil)
	if code != 1 || stdout.Len() != 0 || stderr.String() != "blast-radius: git not found in PATH\n" {
		t.Errorf("exit %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
	}
}

// historyRepo returns a repo whose history has a root commit, an ordinary
// commit with a long subject, a merge and a move:
//
//	root -- edit --------- merge -- move   (main)
//	          \           /
//	           feat1 -- feat2              (feature)
func historyRepo(t *testing.T) string {
	t.Helper()
	dir := newRepo(t)
	write(t, dir, "README.md", lines("readme", 10))
	write(t, dir, "src/core/core.go", lines("core", 40))
	write(t, dir, "src/api/api.go", lines("api", 30))
	commitAll(t, dir, "Initial commit")
	git(t, dir, "tag", "root")

	git(t, dir, "checkout", "-q", "-b", "feature")
	write(t, dir, "src/api/handlers/users.go", lines("users", 80))
	commitAll(t, dir, "Add users handler")
	write(t, dir, "src/api/api.go", lines("api", 20)+lines("route", 15))
	write(t, dir, "docs/api.md", lines("doc", 12))
	commitAll(t, dir, "Document the API")

	git(t, dir, "checkout", "-q", "main")
	write(t, dir, "src/core/core.go", lines("core", 30)+lines("fix", 6))
	write(t, dir, "README.md", lines("readme", 10)+lines("usage", 4))
	commitAll(t, dir, "Fix the core loop so that it no longer spins when the input queue is empty and closed")
	git(t, dir, "tag", "edit")

	git(t, dir, "merge", "-q", "--no-ff", "-m", "Merge branch 'feature'", "feature")
	if err := os.Rename(filepath.Join(dir, "src/core"), filepath.Join(dir, "lib")); err != nil {
		t.Fatal(err)
	}
	commitAll(t, dir, "Move core to lib")
	return dir
}

// A single commit is measured against its parent; the long subject is
// trimmed to fit, and flags may come after the revision.
func TestCommit(t *testing.T) {
	golden(t, "commit", runCLI(t, historyRepo(t), "edit", "--files"))
}

// A root commit is measured against the empty tree.
func TestRootCommit(t *testing.T) {
	golden(t, "commit-root", runCLI(t, historyRepo(t), "root"))
}

// A merge commit shows everything it brought in, against its first parent.
func TestMergeCommit(t *testing.T) {
	golden(t, "commit-merge", runCLI(t, historyRepo(t), "--depth", "1", "HEAD~1", "--files"))
}

// A moved file counts as a delete plus an add.
func TestMovedFile(t *testing.T) {
	golden(t, "commit-move", runCLI(t, historyRepo(t), "HEAD", "--files"))
}

// A..B compares the two endpoints and counts the commits in B not in A.
func TestRange(t *testing.T) {
	golden(t, "range", runCLI(t, historyRepo(t), "edit..feature"))
}

// A...B compares merge-base(A, B) to B and counts only B's side: one commit
// here, where the symmetric difference would count three.
func TestRangeMergeBase(t *testing.T) {
	golden(t, "range-merge-base", runCLI(t, historyRepo(t), "feature...edit", "--files"))
}

func TestBadRevision(t *testing.T) {
	golden(t, "bad-revision", runCLI(t, historyRepo(t), "foo"))
}

func TestRevisionArgs(t *testing.T) {
	dir := historyRepo(t)
	for _, tt := range []struct {
		args []string
		code int
		msg  string
	}{
		{[]string{"HEAD", "root"}, 2, "blast-radius: too many revisions: HEAD root\n"},
		{[]string{"HEAD", "--files", "root"}, 2, "blast-radius: too many revisions: HEAD root\n"},
		{[]string{"src"}, 2, "blast-radius: 'src' is not a revision (use '-- src' to filter by path)\n"},
		{[]string{"HEAD", "--", "src", "--files"}, 0, ""}, // everything after -- is a pathspec
		{[]string{"--", "HEAD", "foo"}, 0, ""},
		{[]string{"HEAD~1.."}, 0, ""},
		{[]string{"...edit"}, 0, ""},
		{[]string{"root..nope"}, 2, "blast-radius: 'root..nope' is not a revision (use '-- root..nope' to filter by path)\n"},
		{[]string{"../src"}, 2, "blast-radius: '../src' is not a revision (use '-- ../src' to filter by path)\n"},
		{[]string{"root^{tree}..HEAD"}, 2, "blast-radius: 'root^{tree}..HEAD' is not a revision"},
	} {
		t.Chdir(dir)
		var stdout, stderr bytes.Buffer
		code := run(tt.args, &stdout, &stderr, testEnv(t))
		if code != tt.code || !strings.HasPrefix(stderr.String(), tt.msg) || (tt.code == 0) != (stderr.Len() == 0) {
			t.Errorf("%q: exit %d, stderr %q", tt.args, code, stderr.String())
		}
	}
}

// Outside a repo, a revision argument fails with git's message, exit 1.
func TestRevisionNotARepo(t *testing.T) {
	got := runCLI(t, t.TempDir(), "HEAD")
	if !strings.Contains(got, "not a git repository") || !strings.HasSuffix(got, "-- exit 1 --\n") {
		t.Errorf("got %q", got)
	}
}

// shapeRepo returns a repo with uncommitted changes spread over nested
// directories, collapsible chains and siblings with equal churn.
func shapeRepo(t *testing.T) string {
	t.Helper()
	dir := newRepo(t)
	write(t, dir, "go.mod", "module example.com/x\n\ngo 1.27\n")
	write(t, dir, "Makefile", lines("make", 5))
	commitAll(t, dir, "initial")

	write(t, dir, "go.mod", "module example.com/x\n\ngo 1.27.1\n")
	write(t, dir, "src/api/handlers/users.go", lines("users", 120))
	write(t, dir, "src/api/handlers/orders.go", lines("orders", 80))
	write(t, dir, "src/api/middleware/auth.go", lines("auth", 40))
	write(t, dir, "src/api/api.go", lines("api", 10))
	write(t, dir, "src/core/core.go", lines("core", 60))
	write(t, dir, "src/internal/util/strings.go", lines("strings", 15))
	write(t, dir, "java/src/main/com/acme/App.java", lines("app", 30))
	write(t, dir, "docs/guide.md", lines("guide", 25))
	write(t, dir, "scripts/build.sh", lines("build", 7))
	write(t, dir, "tools/gen.sh", lines("gen", 7))
	git(t, dir, "add", "-A")
	return dir
}

func TestShapeDefault(t *testing.T) { golden(t, "shape-default", runCLI(t, shapeRepo(t))) }

func TestShapeFiles(t *testing.T) {
	golden(t, "shape-files", runCLI(t, shapeRepo(t), "--files"))
}

func TestShapeDepth1(t *testing.T) {
	golden(t, "shape-depth-1", runCLI(t, shapeRepo(t), "--depth", "1"))
}

func TestShapeDepth0(t *testing.T) {
	// With --files the tree is four levels deep, one more than the default.
	golden(t, "shape-depth-0", runCLI(t, shapeRepo(t), "--depth=0", "--files"))
}

func TestShapeSortName(t *testing.T) {
	golden(t, "shape-sort-name", runCLI(t, shapeRepo(t), "--sort", "name"))
}

// filterRepo is shapeRepo plus untracked files, so filters apply to both
// git diff and git ls-files.
func filterRepo(t *testing.T) string {
	t.Helper()
	dir := shapeRepo(t)
	write(t, dir, "src/api/untracked.go", lines("untracked", 9))
	write(t, dir, "docs/untracked.md", lines("untracked", 4))
	return dir
}

// Pathspecs after -- limit the output and show in the header.
func TestPathspec(t *testing.T) {
	golden(t, "pathspec", runCLI(t, filterRepo(t), "--", "src/"))
}

// --exclude is repeatable and removes matching paths, tracked or untracked.
func TestExclude(t *testing.T) {
	golden(t, "exclude", runCLI(t, filterRepo(t), "--exclude", "src/api/**", "--exclude=*.mod"))
}

// Pathspecs and excludes work together, for ranges too.
func TestRangeFiltered(t *testing.T) {
	golden(t, "range-filtered", runCLI(t, historyRepo(t), "edit..feature", "--exclude", "src/api/handlers/**", "--", "src/"))
}

// Running from a subdirectory gives the same output as from the root, for
// every mode, including excludes relative to the root.
func TestSubdirectory(t *testing.T) {
	for _, tt := range []struct {
		dir  string
		args []string
	}{
		{filterRepo(t), nil},
		{filterRepo(t), []string{"--exclude", ":/docs/**"}},
		{historyRepo(t), []string{"edit"}},
		{historyRepo(t), []string{"root..feature"}},
	} {
		want := runCLI(t, tt.dir, tt.args...)
		if got := runCLI(t, filepath.Join(tt.dir, "src", "api"), tt.args...); got != want {
			t.Errorf("%q from src/api:\n%s\nfrom the root:\n%s", tt.args, got, want)
		}
		if tt.args == nil {
			golden(t, "subdirectory", want)
		}
	}
}

func TestBadFlagValues(t *testing.T) {
	for _, tt := range []struct {
		args []string
		msg  string
	}{
		{[]string{"--sort", "size"}, `blast-radius: invalid --sort "size": must be churn or name`},
		{[]string{"--depth", "-1"}, "blast-radius: invalid --depth -1: must be 0 or more"},
		{[]string{"--depth", "x"}, `invalid value "x" for flag -depth`},
		{[]string{"--color", "yes"}, `blast-radius: invalid --color "yes": must be auto, always or never`},
	} {
		var stdout, stderr bytes.Buffer
		code := run(tt.args, &stdout, &stderr, nil)
		if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), tt.msg) ||
			!strings.Contains(stderr.String(), "-sort") {
			t.Errorf("%v: exit %d, stdout %q, stderr %q", tt.args, code, stdout.String(), stderr.String())
		}
	}
}

// --color=always colors bars and counts, even with NO_COLOR set.
func TestColorAlways(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "src/api/api.go", lines("api", 100))
	write(t, dir, "src/core/core.go", lines("core", 100))
	write(t, dir, "docs/guide.md", lines("guide", 40))
	commitAll(t, dir, "initial")
	write(t, dir, "src/api/api.go", lines("api", 100)+lines("new", 120))  // added only
	write(t, dir, "src/core/core.go", lines("core", 30)+lines("new", 10)) // mostly deleted
	write(t, dir, "docs/guide.md", "")                                    // deleted only
	env := append(testEnv(t), "NO_COLOR=1")
	golden(t, "color-always", runCLIEnv(t, dir, env, "--color=always"))
}

// Piped output stays plain under the default --color=auto.
func TestColorAutoPiped(t *testing.T) {
	t.Chdir(shapeRepo(t))
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr, testEnv(t)); code != 0 || strings.Contains(stdout.String(), "\x1b") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
	}
}

// A narrow terminal truncates long names and keeps the minimum bar width.
func TestNarrow(t *testing.T) {
	golden(t, "narrow", runCLIEnv(t, shapeRepo(t), append(testEnv(t), "COLUMNS=50")))
}

func TestTermWidth(t *testing.T) {
	var buf bytes.Buffer
	for _, tt := range []struct {
		env  []string
		want int
	}{
		{nil, 80},
		{[]string{"COLUMNS=120"}, 120},
		{[]string{"COLUMNS=60", "COLUMNS=70"}, 70},
		{[]string{"COLUMNS="}, 80},
		{[]string{"COLUMNS=abc"}, 80},
		{[]string{"COLUMNS=0"}, 80},
		{[]string{"COLUMNSX=50"}, 80},
	} {
		if got := termWidth(&buf, tt.env); got != tt.want {
			t.Errorf("termWidth(%q) = %d, want %d", tt.env, got, tt.want)
		}
	}
}
