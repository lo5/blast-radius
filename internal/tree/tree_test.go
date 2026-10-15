package tree

import (
	"flag"
	"fmt"
	"strings"
	"testing"
)

// Accept `go test ./... -update`; the golden files live in cmd/blast-radius.
var _ = flag.Bool("update", false, "rewrite golden files")

// dump prints n's subtree as "name +a -d f" lines, indented by depth.
func dump(b *strings.Builder, n *Node, depth int) {
	for _, c := range n.Children {
		name := c.Name
		if c.IsDir {
			name += "/"
		}
		fmt.Fprintf(b, "%s%s +%d -%d %d\n", strings.Repeat("  ", depth), name, c.Added, c.Deleted, c.Files)
		dump(b, c, depth+1)
	}
}

func TestBuild(t *testing.T) {
	root := Build([]FileStat{
		{"go.mod", 4, 3},
		{"src/api/handlers/a.go", 100, 10},
		{"src/api/b.go", 20, 0},
		{"src/core/c.go", 5, 5},
		{"docs/x.md", 1, 0},
		{"src/api/handlers/d.go", 7, 2},
	})
	if !root.IsDir || root.Name != "" {
		t.Errorf("root = %q dir=%v, want unnamed dir", root.Name, root.IsDir)
	}
	if root.Added != 137 || root.Deleted != 20 || root.Files != 6 || root.Churn() != 157 {
		t.Errorf("root totals = +%d -%d %d files", root.Added, root.Deleted, root.Files)
	}
	var b strings.Builder
	dump(&b, root, 0)
	want := `go.mod +4 -3 1
src/ +132 -17 4
  api/ +127 -12 3
    handlers/ +107 -12 2
      a.go +100 -10 1
      d.go +7 -2 1
    b.go +20 -0 1
  core/ +5 -5 1
    c.go +5 -5 1
docs/ +1 -0 1
  x.md +1 -0 1
`
	if got := b.String(); got != want {
		t.Errorf("tree:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildFileAndDirWithSameName(t *testing.T) {
	// A file replaced by a directory shows up as both.
	root := Build([]FileStat{{"a", 0, 3}, {"a/b", 2, 0}})
	var b strings.Builder
	dump(&b, root, 0)
	want := "a +0 -3 1\na/ +2 -0 1\n  b +2 -0 1\n"
	if got := b.String(); got != want {
		t.Errorf("tree:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildEmpty(t *testing.T) {
	root := Build(nil)
	if root.Files != 0 || len(root.Children) != 0 {
		t.Errorf("want empty root, got %+v", root)
	}
}
