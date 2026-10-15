package gitdiff

import (
	"flag"
	"reflect"
	"testing"
)

// Accept `go test ./... -update`; the golden files live in cmd/blast-radius.
var _ = flag.Bool("update", false, "rewrite golden files")

func TestParseNumstat(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []FileStat
	}{
		{"empty", "", nil},
		{"normal", "3\t1\tsrc/a.go\x0010\t0\tREADME.md\x00", []FileStat{
			{Path: "src/a.go", Added: 3, Deleted: 1}, {Path: "README.md", Added: 10, Deleted: 0},
		}},
		{"binary dropped", "-\t-\timg.png\x002\t2\ta.txt\x00", []FileStat{{Path: "a.txt", Added: 2, Deleted: 2}}},
		{"zero-zero dropped", "0\t0\tempty.txt\x001\t0\tb.txt\x00", []FileStat{{Path: "b.txt", Added: 1, Deleted: 0}}},
		{"submodule kept", "1\t1\tvendor/lib\x00", []FileStat{{Path: "vendor/lib", Added: 1, Deleted: 1}}},
		{"space", "1\t2\tmy dir/a file.txt\x00", []FileStat{{Path: "my dir/a file.txt", Added: 1, Deleted: 2}}},
		{"tab", "1\t2\ta\tb.txt\x00", []FileStat{{Path: "a\tb.txt", Added: 1, Deleted: 2}}},
		{"newline", "1\t2\ta\nb.txt\x00", []FileStat{{Path: "a\nb.txt", Added: 1, Deleted: 2}}},
		{"non-ascii", "4\t0\tdocs/héllo 世界.md\x00", []FileStat{{Path: "docs/héllo 世界.md", Added: 4, Deleted: 0}}},
		{"no trailing nul", "1\t0\ta\x002\t0\tb", []FileStat{{Path: "a", Added: 1, Deleted: 0}, {Path: "b", Added: 2, Deleted: 0}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseNumstat([]byte(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseNumstatMalformed(t *testing.T) {
	for _, in := range []string{"garbage\x00", "x\t1\ta\x00"} {
		if _, err := ParseNumstat([]byte(in)); err == nil {
			t.Errorf("ParseNumstat(%q): want error", in)
		}
	}
}
