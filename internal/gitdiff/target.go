package gitdiff

// Diff is what blast-radius measures: a header label and per-file line counts.
type Diff struct {
	Label string
	Files []FileStat
}

// Uncommitted measures staged and unstaged changes to tracked files against HEAD.
func Uncommitted(g Git, pathspecs []string) (Diff, error) {
	if _, err := g.RepoRoot(); err != nil {
		return Diff{}, err
	}
	files, err := g.diff([]string{"HEAD"}, pathspecs)
	if err != nil {
		return Diff{}, err
	}
	return Diff{Label: "working tree vs HEAD", Files: files}, nil
}
