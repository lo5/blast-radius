package gitdiff

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCountLines(t *testing.T) {
	big := strings.Repeat("x\n", 50<<10) // longer than the 8 KiB head and the read buffer
	for _, tt := range []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"trailing newline", "a\nb\n", 2},
		{"no trailing newline", "a\nb", 2},
		{"single line no newline", "a", 1},
		{"blank line", "\n", 1},
		{"binary", "a\x00b\n", 0},
		{"large", big, 50 << 10},
		{"large no trailing newline", big + "x", 50<<10 + 1},
		{"NUL after 8 KiB is text", strings.Repeat("x", 8<<10) + "\x00\n", 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := countLines(strings.NewReader(tt.in))
			if err != nil || got != tt.want {
				t.Errorf("got %d, %v; want %d", got, err, tt.want)
			}
		})
	}
}

func TestCountFile(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	text := write("text.txt", "a\nb\nc")
	link := filepath.Join(dir, "link")
	if err := os.Symlink(text, link); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		path string
		want int
	}{
		{text, 3},
		{write("empty.txt", ""), 0},
		{write("bin.dat", "\x00\x01\x02"), 0},
		{link, 0}, // symlinks are skipped, not followed
		{dir, 0},  // not a regular file
	} {
		if got, err := countFile(tt.path); err != nil || got != tt.want {
			t.Errorf("countFile(%s) = %d, %v; want %d", filepath.Base(tt.path), got, err, tt.want)
		}
	}
}

func TestCountFileUnreadable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can read anything")
	}
	p := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(p, []byte("a\n"), 0o000); err != nil {
		t.Fatal(err)
	}
	if _, err := countFile(p); err == nil {
		t.Error("want error")
	}
}

func TestUntracked(t *testing.T) {
	env := []string{"PATH=" + os.Getenv("PATH"), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null"}
	root := t.TempDir()
	run := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir, cmd.Env = dir, env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(name, content string, perm os.FileMode) {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), perm); err != nil {
			t.Fatal(err)
		}
	}
	run(root, "init", "-q")
	write(".gitignore", "*.log\n", 0o644)
	write("a/new.txt", "1\n2\n", 0o644)
	write("a/ignored.log", "x\n", 0o644)
	write("a/empty.txt", "", 0o644)
	write("img.png", "\x89PNG\x00", 0o644)
	write("nested/.keep", "", 0o644)
	run(filepath.Join(root, "nested"), "init", "-q")
	write("nested/inner.txt", "inner\n", 0o644)
	if err := os.Symlink("a/new.txt", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	unreadable := os.Getuid() != 0
	if unreadable {
		write("secret.txt", "s\n", 0o000)
	}

	t.Chdir(filepath.Join(root, "a")) // lists the whole repo, not just the current directory
	files, warnings, err := Git{Env: env}.Untracked(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []FileStat{{Path: ".gitignore", Added: 1}, {Path: "a/new.txt", Added: 2}}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("files = %+v, want %+v", files, want)
	}
	var wantWarn []string
	if unreadable {
		wantWarn = []string{"cannot read secret.txt: permission denied"}
	}
	if !reflect.DeepEqual(warnings, wantWarn) {
		t.Errorf("warnings = %q, want %q", warnings, wantWarn)
	}

	// Only excludes: still the whole repo, minus the excluded paths.
	files, _, err = Git{Env: env}.Untracked(root, Pathspecs(nil, []string{":/.gitignore"}))
	if err != nil {
		t.Fatal(err)
	}
	if want := []FileStat{{Path: "a/new.txt", Added: 2}}; !reflect.DeepEqual(files, want) {
		t.Errorf("with excludes: files = %+v, want %+v", files, want)
	}
}
