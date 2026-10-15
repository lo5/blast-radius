package gitdiff

import (
	"errors"
	"fmt"
	"strings"
)

// Diff is what blast-radius measures: a header label and per-file line counts.
// Subject is the commit subject for a single commit, else empty.
// Warnings are non-fatal problems, such as untracked files that can't be read.
type Diff struct {
	Label    string
	Subject  string
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
// empty, else a single commit.
func Measure(g Git, arg string, pathspecs []string) (Diff, error) {
	switch Classify(arg) {
	case KindUncommitted:
		return Uncommitted(g, pathspecs)
	case KindCommit:
		return Commit(g, arg, pathspecs)
	}
	return Diff{}, errors.New("commit ranges are not supported yet")
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
