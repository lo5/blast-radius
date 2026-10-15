package render

import (
	"flag"
	"strings"
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

// renderTree is a small tree used by the Render tests.
func renderTree() *tree.Node {
	return tree.Build([]tree.FileStat{
		{Path: "src/api/handlers/a.go", Added: 1000, Deleted: 90},
		{Path: "src/api/b.go", Added: 20, Deleted: 0},
		{Path: "src/core/c.go", Added: 5, Deleted: 300},
		{Path: "docs/x.md", Added: 1, Deleted: 0},
		{Path: "go.mod", Added: 4, Deleted: 3},
	})
}

func TestRender(t *testing.T) {
	got := Render("working tree vs HEAD", renderTree(), 100)
	want := `working tree vs HEAD · 5 files · +1,030 -393

src/             ████████████████████████████████████████  +1,025 -390  3 files  99%
├─ api/          ███████████████████████████████▍          +1,020  -90  2 files  78%
│  ├─ handlers/  ██████████████████████████████▉           +1,000  -90   1 file  77%
│  │  └─ a.go    ██████████████████████████████▉           +1,000  -90   1 file  77%
│  └─ b.go       ▋                                            +20   -0   1 file   1%
└─ core/         ████████▋                                     +5 -300   1 file  21%
   └─ c.go       ████████▋                                     +5 -300   1 file  21%
docs/            ▏                                             +1   -0   1 file  <1%
└─ x.md          ▏                                             +1   -0   1 file  <1%
go.mod           ▎                                             +4   -3   1 file  <1%
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// A narrow terminal caps the name column at 40% of the width, truncates
// names but keeps tree prefixes, and keeps the minimum bar width.
func TestRenderNarrow(t *testing.T) {
	got := Render("working tree vs HEAD", renderTree(), 30)
	want := `working tree vs HEAD · 5 files · +1,030 -393

src/          ██████████  +1,025 -390  3 files  99%
├─ api/       ███████▉    +1,020  -90  2 files  78%
│  ├─ handl…  ███████▊    +1,000  -90   1 file  77%
│  │  └─ a.…  ███████▊    +1,000  -90   1 file  77%
│  └─ b.go    ▏              +20   -0   1 file   1%
└─ core/      ██▏             +5 -300   1 file  21%
   └─ c.go    ██▏             +5 -300   1 file  21%
docs/         ▏               +1   -0   1 file  <1%
└─ x.md       ▏               +1   -0   1 file  <1%
go.mod        ▏               +4   -3   1 file  <1%
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// The bar width fills the rest of the line, within [10, 40].
func TestRenderBarWidth(t *testing.T) {
	root := tree.Build([]tree.FileStat{{Path: "a/x", Added: 1}})
	// Without its bar, a row is 27 wide: name column "└─ x", "+1 -0",
	// "1 file", "100%" and four separators.
	for termWidth, want := range map[int]int{20: 10, 37: 10, 38: 11, 45: 18, 66: 39, 67: 40, 200: 40} {
		out := Render("l", root, termWidth)
		row := strings.Split(out, "\n")[2]
		if got := width(row) - 27; got != want {
			t.Errorf("width %d: bar width %d, want %d\n%s", termWidth, got, want, row)
		}
	}
}

func TestRenderNoChanges(t *testing.T) {
	got := Render("working tree vs HEAD", tree.Build(nil), 100)
	want := "working tree vs HEAD · 0 files · +0 -0\n\nno changes\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEighths(t *testing.T) {
	tests := []struct {
		churn, maxChurn, w, want int
	}{
		{100, 100, 40, 320}, // the largest row fills the width
		{100, 100, 10, 80},
		{50, 100, 10, 40},
		{1, 1000000, 40, 1}, // tiny but non-zero: at least 1 eighth
		{0, 100, 40, 0},
		{1, 128, 10, 1},  // 0.625 rounds up
		{3, 1280, 10, 1}, // 0.1875: minimum
		{3, 64, 10, 4},   // 3.75 rounds up
	}
	for _, tt := range tests {
		if got := Eighths(tt.churn, tt.maxChurn, tt.w); got != tt.want {
			t.Errorf("Eighths(%d, %d, %d) = %d, want %d", tt.churn, tt.maxChurn, tt.w, got, tt.want)
		}
	}
}

func TestBar(t *testing.T) {
	for eighths, want := range map[int]string{
		0: "", 1: "▏", 2: "▎", 3: "▍", 4: "▌", 5: "▋", 6: "▊", 7: "▉",
		8: "█", 9: "█▏", 23: "██▉", 80: "██████████",
	} {
		if got := Bar(eighths); got != want {
			t.Errorf("Bar(%d) = %q, want %q", eighths, got, want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		prefix, label string
		w             int
		want          string
	}{
		{"", "src/", 10, "src/"},
		{"", "internal/util/", 14, "internal/util/"},
		{"", "internal/util/", 13, "internal/uti…"},
		{"│  ├─ ", "handlers/", 12, "│  ├─ handl…"},
		{"", "ünïcödé/", 5, "ünïc…"},
		{"│  │  └─ ", "a.go", 6, "│  │  └─ …"}, // the prefix is always kept
	}
	for _, tt := range tests {
		if got := truncate(tt.prefix, tt.label, tt.w); got != tt.want {
			t.Errorf("truncate(%q, %q, %d) = %q, want %q", tt.prefix, tt.label, tt.w, got, tt.want)
		}
	}
}
