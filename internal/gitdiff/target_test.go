package gitdiff

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestClassify(t *testing.T) {
	for arg, want := range map[string]Kind{
		"":            KindUncommitted,
		"HEAD~1":      KindCommit,
		"abc123":      KindCommit,
		"v1.2":        KindCommit,
		"main..feat":  KindRange,
		"HEAD~5..":    KindRange,
		"..main":      KindRange,
		"v1.2...HEAD": KindMergeBase,
		"main...":     KindMergeBase,
	} {
		if got := Classify(arg); got != want {
			t.Errorf("Classify(%q) = %d, want %d", arg, got, want)
		}
	}
}

func TestNotRevisionError(t *testing.T) {
	want := "'foo' is not a revision (use '-- foo' to filter by path)"
	if got := (NotRevisionError{"foo"}).Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCommit(t *testing.T) {
	env := []string{
		"PATH=" + os.Getenv("PATH"), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_AUTHOR_DATE=2026-01-01T00:00:00Z",
		"GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@example.com", "GIT_COMMITTER_DATE=2026-01-01T00:00:00Z",
	}
	root := t.TempDir()
	git := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir, cmd.Env = root, env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	commit := func(path, content, msg string) {
		if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		git("add", "-A")
		git("commit", "-q", "--allow-empty-message", "-m", msg)
	}
	git("init", "-q")
	commit("a.txt", "1\n2\n", "First")
	commit("a.txt", "1\n3\n4\n", "Second one")
	commit("a.txt", "1\n", "")
	t.Chdir(root)
	g := Git{Env: env}

	for _, tt := range []struct {
		rev  string
		want Diff
	}{
		{"HEAD~2", Diff{Subject: "First", Files: []FileStat{{Path: "a.txt", Added: 2}}}}, // root commit
		{"HEAD~1", Diff{Subject: "Second one", Files: []FileStat{{Path: "a.txt", Added: 2, Deleted: 1}}}},
		{"HEAD", Diff{Files: []FileStat{{Path: "a.txt", Deleted: 2}}}}, // empty subject
	} {
		got, err := Commit(g, tt.rev, nil)
		if err != nil {
			t.Fatalf("%s: %v", tt.rev, err)
		}
		short := git("log", "-1", "--format=%h", tt.rev)
		tt.want.Label = short[:len(short)-1]
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: got %+v, want %+v", tt.rev, got, tt.want)
		}
	}

	_, err := Commit(g, "nope", nil)
	if notRev := (NotRevisionError{}); !errors.As(err, &notRev) || notRev.Arg != "nope" {
		t.Errorf("got %v, want NotRevisionError", err)
	}
	// A tree is not a commit.
	if _, err := Commit(g, "HEAD^{tree}", nil); !errors.As(err, new(NotRevisionError)) {
		t.Errorf("tree: got %v, want NotRevisionError", err)
	}
}
