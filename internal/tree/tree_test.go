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

// shaped builds a tree from files, applies fn and dumps the result.
func shaped(files []FileStat, fn func(*Node)) string {
	root := Build(files)
	fn(root)
	var b strings.Builder
	dump(&b, root, 0)
	return b.String()
}

func check(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("tree:\n%s\nwant:\n%s", got, want)
	}
}

func TestHideFiles(t *testing.T) {
	got := shaped([]FileStat{
		{"go.mod", 4, 3},
		{"src/a.go", 10, 0},
		{"src/api/b.go", 5, 1},
		{"docs/x.md", 1, 0},
	}, HideFiles)
	// Root files stay; directories keep totals of their hidden files.
	check(t, got, `go.mod +4 -3 1
src/ +15 -1 2
  api/ +5 -1 1
docs/ +1 -0 1
`)
}

func TestCollapse(t *testing.T) {
	got := shaped([]FileStat{
		{"go.mod", 1, 0},
		{"src/main/java/com/acme/App.java", 9, 0},
		{"internal/util/u.go", 3, 0},
		{"internal/util/v.go", 2, 0},
		{"pkg/a/b/c.go", 1, 1},
		{"pkg/d.go", 1, 0},
	}, Collapse)
	// Chains repeat until stable; a chain stops at a file (here with all
	// files shown); a directory with a visible file doesn't merge.
	check(t, got, `go.mod +1 -0 1
src/main/java/com/acme/ +9 -0 1
  App.java +9 -0 1
internal/util/ +5 -0 2
  u.go +3 -0 1
  v.go +2 -0 1
pkg/ +2 -1 2
  a/b/ +1 -1 1
    c.go +1 -1 1
  d.go +1 -0 1
`)
}

func TestCollapseAfterHideFiles(t *testing.T) {
	got := shaped([]FileStat{
		{"a/b/c/x.go", 1, 0},
		{"lib/lib.go", 7, 0},
		{"lib/sub/s.go", 2, 0},
		{"root.go", 1, 1},
	}, func(n *Node) { HideFiles(n); Collapse(n) })
	// A directory with only hidden files beside one subdirectory merges and
	// keeps its own totals. A lone root directory child is not merged into
	// the root.
	check(t, got, `a/b/c/ +1 -0 1
lib/sub/ +9 -0 2
root.go +1 -1 1
`)
	got = shaped([]FileStat{{"only/dir/x.go", 1, 0}}, func(n *Node) { HideFiles(n); Collapse(n) })
	check(t, got, "only/dir/ +1 -0 1\n")
}

func TestSortChurn(t *testing.T) {
	got := shaped([]FileStat{
		{"b.go", 1, 1},
		{"z/x.go", 50, 0},
		{"a/x.go", 1, 1},
		{"m/small.go", 1, 0},
		{"m/big.go", 0, 9},
		{"c.go", 100, 0},
	}, func(n *Node) { Sort(n, false) })
	// Highest churn first, ties by name; directories and files mixed.
	check(t, got, `c.go +100 -0 1
z/ +50 -0 1
  x.go +50 -0 1
m/ +1 -9 2
  big.go +0 -9 1
  small.go +1 -0 1
a/ +1 -1 1
  x.go +1 -1 1
b.go +1 -1 1
`)
}

func TestSortName(t *testing.T) {
	got := shaped([]FileStat{
		{"b.go", 1, 1},
		{"z/x.go", 50, 0},
		{"a/y.go", 1, 1},
		{"a/x.go", 9, 1},
		{"c.go", 100, 0},
	}, func(n *Node) { Sort(n, true) })
	check(t, got, `a/ +10 -2 2
  x.go +9 -1 1
  y.go +1 -1 1
b.go +1 -1 1
c.go +100 -0 1
z/ +50 -0 1
  x.go +50 -0 1
`)
}

func TestLimitDepth(t *testing.T) {
	files := []FileStat{
		{"go.mod", 1, 0},
		{"src/api/handlers/a.go", 4, 0},
		{"src/main/java/com/App.java", 2, 0},
	}
	shape := func(depth int) func(*Node) {
		return func(n *Node) { HideFiles(n); Collapse(n); LimitDepth(n, depth) }
	}
	check(t, shaped(files, shape(1)), "go.mod +1 -0 1\nsrc/ +6 -0 2\n")
	// Collapsed chains take up one level.
	want := `go.mod +1 -0 1
src/ +6 -0 2
  api/handlers/ +4 -0 1
  main/java/com/ +2 -0 1
`
	check(t, shaped(files, shape(2)), want)
	check(t, shaped(files, shape(0)), want)
}

func TestShape(t *testing.T) {
	files := []FileStat{
		{"go.mod", 1, 0},
		{"src/api/a.go", 4, 0},
		{"src/api/b.go", 1, 0},
		{"src/core/deep/c.go", 9, 0},
	}
	shape := func(o Options) func(*Node) { return func(n *Node) { Shape(n, o) } }
	check(t, shaped(files, shape(Options{Depth: 3})), `src/ +14 -0 3
  core/deep/ +9 -0 1
  api/ +5 -0 2
go.mod +1 -0 1
`)
	check(t, shaped(files, shape(Options{Files: true, ByName: true, Depth: 2})), `go.mod +1 -0 1
src/ +14 -0 3
  api/ +5 -0 2
  core/deep/ +9 -0 1
`)
}
