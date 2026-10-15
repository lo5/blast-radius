// Package gitdiff runs git and turns its output into per-file line counts.
// All git and filesystem I/O lives here.
package gitdiff

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

// diffFlags are added to every diff so user config and diff drivers can't
// change the output, and paths are relative to the repo root.
var diffFlags = []string{
	"--numstat", "-z", "--no-renames", "--no-ext-diff", "--no-textconv", "--no-color", "--no-relative",
}

// Git runs git commands in the current directory.
type Git struct {
	Env []string // environment for git; nil inherits the process environment
}

// Run runs git with args and returns its stdout. If git fails, the error
// message is git's trimmed stderr.
func (g Git) Run(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Env = g.Env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, errors.New("git not found in PATH")
		}
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, errors.New(msg)
		}
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return stdout.Bytes(), nil
}

// RepoRoot returns the absolute path of the repository's top-level directory.
// It fails with git's message when not inside a repository.
func (g Git) RepoRoot() (string, error) {
	out, err := g.Run("rev-parse", "--show-toplevel")
	return strings.TrimSpace(string(out)), err
}

// EmptyTree returns the hash of the empty tree in the repo's hash format, so
// SHA-256 repositories work too.
func (g Git) EmptyTree() (string, error) {
	out, err := g.Run("hash-object", "-t", "tree", "--stdin")
	return strings.TrimSpace(string(out)), err
}

// Pathspecs returns paths followed by one `:(exclude,glob)GLOB` pathspec per
// exclude, so git does all the glob matching. Like other pathspecs, a glob is
// relative to the current directory unless it starts with ":/".
func Pathspecs(paths, excludes []string) []string {
	specs := slices.Clone(paths)
	for _, glob := range excludes {
		if rest, ok := strings.CutPrefix(glob, ":/"); ok {
			specs = append(specs, ":(exclude,glob,top)"+rest)
		} else {
			specs = append(specs, ":(exclude,glob)"+glob)
		}
	}
	return specs
}

// isExclude reports whether pathspec excludes paths rather than selecting them.
func isExclude(pathspec string) bool {
	if magic, ok := strings.CutPrefix(pathspec, ":("); ok {
		magic, _, _ = strings.Cut(magic, ")")
		return slices.Contains(strings.Split(magic, ","), "exclude")
	}
	return strings.HasPrefix(pathspec, ":!") || strings.HasPrefix(pathspec, ":^")
}

// diff runs `git diff <revs> <diffFlags> -- <pathspecs>` and parses the result.
func (g Git) diff(revs, pathspecs []string) ([]FileStat, error) {
	args := append([]string{"diff"}, revs...)
	args = append(args, diffFlags...)
	args = append(args, "--")
	args = append(args, pathspecs...)
	out, err := g.Run(args...)
	if err != nil {
		return nil, err
	}
	return ParseNumstat(out)
}
