package render

import (
	"flag"
	"regexp"
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
	got := Render("working tree vs HEAD", "", renderTree(), 100, false)
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
	got := Render("working tree vs HEAD", "", renderTree(), 30, false)
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
		out := Render("l", "", root, termWidth, false)
		row := strings.Split(out, "\n")[2]
		if got := width(row) - 27; got != want {
			t.Errorf("width %d: bar width %d, want %d\n%s", termWidth, got, want, row)
		}
	}
}

func TestRenderNoChanges(t *testing.T) {
	got := Render("working tree vs HEAD", "", tree.Build(nil), 100, false)
	want := "working tree vs HEAD · 0 files · +0 -0\n\nno changes\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Header labels: uncommitted, and a commit whose subject is trimmed to fit.
func TestHeader(t *testing.T) {
	root := tree.Build([]tree.FileStat{{Path: "a.go", Added: 1200, Deleted: 3}})
	for _, tt := range []struct {
		label, subject string
		width          int
		want           string
	}{
		{"working tree vs HEAD", "", 10, "working tree vs HEAD · 1 file · +1,200 -3"},
		{"abc1234", "Fix the parser", 80, "abc1234 Fix the parser · 1 file · +1,200 -3"},
		{"abc1234", "Fix the parser", 43, "abc1234 Fix the parser · 1 file · +1,200 -3"},
		{"abc1234", "Fix the parser", 42, "abc1234 Fix the pars… · 1 file · +1,200 -3"},
		{"abc1234", "Fix the parser", 30, "abc1234 … · 1 file · +1,200 -3"},
		{"abc1234", "Fix the parser", 10, "abc1234 … · 1 file · +1,200 -3"},
	} {
		if got := Header(tt.label, tt.subject, root, tt.width); got != tt.want {
			t.Errorf("Header(%q, %q, %d) = %q, want %q", tt.label, tt.subject, tt.width, got, tt.want)
		}
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

func TestGreenCells(t *testing.T) {
	tests := []struct {
		n, added, churn, want int
	}{
		{10, 10, 10, 10}, // all added
		{10, 0, 10, 0},   // all deleted
		{10, 5, 10, 5},
		{10, 1, 3, 3},  // 3.33 rounds down
		{10, 2, 3, 7},  // 6.67 rounds up
		{1, 1, 2, 1},   // 0.5 rounds up: a one-cell bar goes green
		{3, 1, 6, 1},   // 0.5 rounds up
		{1, 1, 3, 0},   // 0.33 rounds down
		{4, 1, 100, 0}, // a sliver of added lines gets no cell
		{0, 1, 1, 0},
		{5, 0, 0, 0},
	}
	for _, tt := range tests {
		if got := GreenCells(tt.n, tt.added, tt.churn); got != tt.want {
			t.Errorf("GreenCells(%d, %d, %d) = %d, want %d", tt.n, tt.added, tt.churn, got, tt.want)
		}
	}
}

func TestColorBar(t *testing.T) {
	const g, r, end = "\x1b[32m", "\x1b[31m", "\x1b[0m"
	tests := []struct {
		bar          string
		added, churn int
		want         string
	}{
		{"███▍", 1, 2, g + "██" + end + r + "█▍" + end},
		{"███▍", 3, 4, g + "███" + end + r + "▍" + end},
		{"███▍", 4, 4, g + "███▍" + end}, // the partial cell takes its side's color
		{"███▍", 0, 4, r + "███▍" + end},
		{"▏", 1, 1, g + "▏" + end},
		{"", 0, 0, ""},
	}
	for _, tt := range tests {
		if got := colorBar(tt.bar, tt.added, tt.churn); got != tt.want {
			t.Errorf("colorBar(%q, %d, %d) = %q, want %q", tt.bar, tt.added, tt.churn, got, tt.want)
		}
	}
}

func TestUseColor(t *testing.T) {
	tests := []struct {
		mode    string
		tty     bool
		noColor string
		want    bool
	}{
		{"auto", true, "", true},
		{"auto", true, "1", false},
		{"auto", false, "", false},
		{"always", false, "1", true}, // NO_COLOR doesn't override always
		{"never", true, "", false},
	}
	for _, tt := range tests {
		if got := UseColor(tt.mode, tt.tty, tt.noColor); got != tt.want {
			t.Errorf("UseColor(%q, %v, %q) = %v, want %v", tt.mode, tt.tty, tt.noColor, got, tt.want)
		}
	}
}

// With color, padding counts cells, so colored rows line up with plain ones.
func TestRenderColor(t *testing.T) {
	plain := Render("l", "", renderTree(), 100, false)
	colored := Render("l", "", renderTree(), 100, true)
	if !strings.Contains(colored, "\x1b[32m+1,025\x1b[0m \x1b[31m-390\x1b[0m") {
		t.Errorf("counts not colored:\n%s", colored)
	}
	stripped := regexp.MustCompile("\x1b\\[[0-9]+m").ReplaceAllString(colored, "")
	if stripped != plain {
		t.Errorf("colored output without SGR codes differs from plain:\n%s\nwant:\n%s", stripped, plain)
	}
}
