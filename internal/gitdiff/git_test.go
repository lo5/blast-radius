package gitdiff

import (
	"reflect"
	"strings"
	"testing"
)

func TestRunReturnsGitStderr(t *testing.T) {
	_, err := Git{}.Run("rev-parse", "--verify", "--not-a-real-flag-xyz")
	if err == nil || strings.HasPrefix(err.Error(), "git ") || err.Error() != strings.TrimSpace(err.Error()) {
		t.Fatalf("want trimmed git stderr, got %v", err)
	}
}

func TestRunGitNotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := Git{}.Run("version")
	if err == nil || err.Error() != "git not found in PATH" {
		t.Fatalf("got %v", err)
	}
}

func TestEmptyTree(t *testing.T) {
	t.Chdir(t.TempDir())
	got, err := Git{Env: []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null"}}.EmptyTree()
	if err != nil {
		t.Fatal(err)
	}
	if want := "4b825dc642cb6eb9a060e54bf8d69288fbee4904"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPathspecs(t *testing.T) {
	got := Pathspecs([]string{"src/", "."}, []string{"*.md", ":/docs/**"})
	want := []string{"src/", ".", ":(exclude,glob)*.md", ":(exclude,glob,top)docs/**"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := Pathspecs(nil, nil); len(got) != 0 {
		t.Errorf("got %q, want none", got)
	}
}

func TestIsExclude(t *testing.T) {
	for spec, want := range map[string]bool{
		"src/":                false,
		":/":                  false,
		":(glob)**/*.go":      false,
		":(exclude,glob)*.md": true,
		":(top,exclude)docs":  true,
		":!vendor":            true,
		":^vendor":            true,
		":(excludes)x":        false,
		"(exclude)not-magic":  false,
	} {
		if got := isExclude(spec); got != want {
			t.Errorf("isExclude(%q) = %v, want %v", spec, got, want)
		}
	}
}
