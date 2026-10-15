package render

import (
	"flag"
	"testing"

	"github.com/lo5/blast-radius/internal/tree"
)

// Accept `go test ./... -update`; the golden files live in cmd/blast-radius.
var _ = flag.Bool("update", false, "rewrite golden files")

func TestThousands(t *testing.T) {
	for n, want := range map[int]string{
		0: "0", 7: "7", 999: "999", 1000: "1,000", 1204: "1,204",
		999999: "999,999", 1000000: "1,000,000", 12345678: "12,345,678",
	} {
		if got := Thousands(n); got != want {
			t.Errorf("Thousands(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestPlural(t *testing.T) {
	for n, want := range map[int]string{0: "0 files", 1: "1 file", 2: "2 files", 1500: "1,500 files"} {
		if got := Plural(n, "file"); got != want {
			t.Errorf("Plural(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestPercent(t *testing.T) {
	tests := []struct {
		churn, total int
		want         string
	}{
		{100, 100, "100%"},
		{82, 100, "82%"},
		{1, 200, "1%"}, // 0.5 rounds up
		{1, 201, "<1%"},
		{1, 1000, "<1%"},
		{0, 1000, "0%"},
		{2, 3, "67%"},
	}
	for _, tt := range tests {
		if got := Percent(tt.churn, tt.total); got != tt.want {
			t.Errorf("Percent(%d, %d) = %q, want %q", tt.churn, tt.total, got, tt.want)
		}
	}
}

func TestRender(t *testing.T) {
	root := tree.Build([]tree.FileStat{
		{Path: "src/api/handlers/a.go", Added: 1000, Deleted: 90},
		{Path: "src/api/b.go", Added: 20, Deleted: 0},
		{Path: "src/core/c.go", Added: 5, Deleted: 300},
		{Path: "docs/x.md", Added: 1, Deleted: 0},
		{Path: "go.mod", Added: 4, Deleted: 3},
	})
	got := Render("working tree vs HEAD", root)
	want := `working tree vs HEAD · 5 files · +1,030 -393

src/             +1,025 -390  3 files  99%
├─ api/          +1,020  -90  2 files  78%
│  ├─ handlers/  +1,000  -90   1 file  77%
│  │  └─ a.go    +1,000  -90   1 file  77%
│  └─ b.go          +20   -0   1 file   1%
└─ core/             +5 -300   1 file  21%
   └─ c.go           +5 -300   1 file  21%
docs/                +1   -0   1 file  <1%
└─ x.md              +1   -0   1 file  <1%
go.mod               +4   -3   1 file  <1%
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderNoChanges(t *testing.T) {
	got := Render("working tree vs HEAD", tree.Build(nil))
	want := "working tree vs HEAD · 0 files · +0 -0\n\nno changes\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
