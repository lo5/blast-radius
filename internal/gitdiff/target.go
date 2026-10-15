package gitdiff

// Diff is what blast-radius measures: a header label and per-file line counts.
// Warnings are non-fatal problems, such as untracked files that can't be read.
type Diff struct {
	Label    string
	Files    []FileStat
	Warnings []string
}

// Uncommitted measures staged, unstaged and untracked changes against HEAD,
// or against the empty tree when HEAD is unborn.
func Uncommitted(g Git, pathspecs []string) (Diff, error) {
	root, err := g.RepoRoot()
	if err != nil {
		return Diff{}, err
	}
	base := "HEAD"
	if _, err := g.Run("rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		if base, err = g.EmptyTree(); err != nil {
			return Diff{}, err
		}
	}
	files, err := g.diff([]string{base}, pathspecs)
	if err != nil {
		return Diff{}, err
	}
	untracked, warnings, err := g.Untracked(root, pathspecs)
	if err != nil {
		return Diff{}, err
	}
	return Diff{Label: "working tree vs HEAD", Files: append(files, untracked...), Warnings: warnings}, nil
}
