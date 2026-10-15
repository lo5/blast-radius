package gitdiff

import (
	"fmt"
	"strconv"
	"strings"
)

// Diff is what blast-radius measures: a header label and per-file line counts.
// Subject is the commit subject for a single commit, else empty. Range is
// set for commit ranges, and Commits is then the number of commits in it.
// Warnings are non-fatal problems, such as untracked files that can't be read.
type Diff struct {
	Label    string
	Subject  string
	Range    bool
	Commits  int
	Files    []FileStat
	Warnings []string
}

// Kind is what a revision argument asks to measure.
type Kind int

const (
	KindUncommitted Kind = iota // no argument
	KindCommit                  // a single revision
	KindRange                   // A..B
	KindMergeBase               // A...B
)

// Classify returns what arg measures. Any argument containing ".." is a range.
func Classify(arg string) Kind {
	switch {
	case arg == "":
		return KindUncommitted
	case strings.Contains(arg, "..."):
		return KindMergeBase
	case strings.Contains(arg, ".."):
		return KindRange
	}
	return KindCommit
}

// NotRevisionError reports an argument that doesn't name a commit.
type NotRevisionError struct{ Arg string }

func (e NotRevisionError) Error() string {
	return fmt.Sprintf("'%s' is not a revision (use '-- %s' to filter by path)", e.Arg, e.Arg)
}

// Measure returns the diff that arg asks for: uncommitted changes when arg is
// empty, a range when it contains "..", else a single commit.
func Measure(g Git, arg string, pathspecs []string) (Diff, error) {
	switch Classify(arg) {
	case KindUncommitted:
		return Uncommitted(g, pathspecs)
	case KindCommit:
		return Commit(g, arg, pathspecs)
	}
	return Range(g, arg, pathspecs)
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

// Commit measures one commit against its first parent, or against the empty
// tree for a root commit. It returns a NotRevisionError if rev isn't a commit.
func Commit(g Git, rev string, pathspecs []string) (Diff, error) {
	if _, err := g.RepoRoot(); err != nil { // not a repo: git's message, not ours
		return Diff{}, err
	}
	out, err := g.Run("rev-parse", "--verify", "--quiet", rev+"^{commit}")
	if err != nil {
		return Diff{}, NotRevisionError{rev}
	}
	sha := strings.TrimSpace(string(out))
	if out, err = g.Run("rev-list", "--parents", "-n", "1", sha); err != nil {
		return Diff{}, err
	}
	var base string
	if parents := strings.Fields(string(out))[1:]; len(parents) > 0 {
		base = parents[0]
	} else if base, err = g.EmptyTree(); err != nil {
		return Diff{}, err
	}
	files, err := g.diff([]string{base, sha}, pathspecs)
	if err != nil {
		return Diff{}, err
	}
	if out, err = g.Run("log", "-1", "--format=%h %s", sha); err != nil {
		return Diff{}, err
	}
	label, subject, _ := strings.Cut(strings.TrimRight(string(out), "\n"), " ")
	return Diff{Label: label, Subject: subject, Files: files}, nil
}

// Range measures a commit range, A..B or A...B, passed to git diff as typed.
// An empty side means HEAD, as in git. It returns a NotRevisionError if
// either side isn't a commit.
func Range(g Git, arg string, pathspecs []string) (Diff, error) {
	if _, err := g.RepoRoot(); err != nil { // not a repo: git's message, not ours
		return Diff{}, err
	}
	sep := ".."
	if Classify(arg) == KindMergeBase {
		sep = "..."
	}
	a, b, _ := strings.Cut(arg, sep)
	for _, side := range []*string{&a, &b} {
		if *side == "" {
			*side = "HEAD"
		}
		if _, err := g.Run("rev-parse", "--verify", "--quiet", *side+"^{commit}"); err != nil {
			return Diff{}, NotRevisionError{arg}
		}
	}
	files, err := g.diff([]string{arg}, pathspecs)
	if err != nil {
		return Diff{}, err
	}
	// Count the commits on B's side only: rev-list A...B would also count A's.
	count := arg
	if sep == "..." {
		out, err := g.Run("merge-base", a, b)
		if err != nil {
			return Diff{}, err
		}
		count = strings.TrimSpace(string(out)) + ".." + b
	}
	out, err := g.Run("rev-list", "--count", count)
	if err != nil {
		return Diff{}, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return Diff{}, fmt.Errorf("git rev-list --count: %w", err)
	}
	return Diff{Label: arg, Range: true, Commits: n, Files: files}, nil
}
