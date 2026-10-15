package gitdiff

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/lo5/blast-radius/internal/tree"
)

// FileStat is the line count for one changed file.
type FileStat = tree.FileStat

// ParseNumstat parses `git diff --numstat -z --no-renames` output, where each
// record is "<added>\t<deleted>\t<path>\0". Binary records ("-") and records
// with no changed lines are dropped.
func ParseNumstat(data []byte) ([]FileStat, error) {
	var files []FileStat
	for rec := range bytes.SplitSeq(data, []byte{0}) {
		if len(rec) == 0 {
			continue
		}
		parts := strings.SplitN(string(rec), "\t", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("unexpected numstat record %q", rec)
		}
		if parts[0] == "-" || parts[1] == "-" {
			continue
		}
		added, err1 := strconv.Atoi(parts[0])
		deleted, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("unexpected numstat record %q", rec)
		}
		if added == 0 && deleted == 0 {
			continue
		}
		files = append(files, FileStat{Path: parts[2], Added: added, Deleted: deleted})
	}
	return files, nil
}
