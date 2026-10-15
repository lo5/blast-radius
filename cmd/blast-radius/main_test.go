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
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	code := run(append([]string{"--color=never"}, args...), &stdout, &stderr, testEnv(t))
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

	golden(t, "uncommitted", runCLI(t, dir))
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

func TestUnexpectedArgument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"foo"}, &stdout, &stderr, nil); code != 2 || stdout.Len() != 0 {
		t.Errorf("exit %d, stdout %q", code, stdout.String())
	}
}
