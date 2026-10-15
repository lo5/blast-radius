// Command blast-radius shows how many lines changed in each directory of a
// git repository.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/lo5/blast-radius/internal/gitdiff"
	"github.com/lo5/blast-radius/internal/render"
	"github.com/lo5/blast-radius/internal/tree"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Environ()))
}

// run is the whole CLI: it parses args, collects all data, then writes the
// output in one go and returns the exit code. env is the environment for git
// and for settings such as COLUMNS.
func run(args []string, stdout, stderr io.Writer, env []string) int {
	fs := flag.NewFlagSet("blast-radius", flag.ContinueOnError)
	fs.SetOutput(stderr)
	color := fs.String("color", "auto", "use color: auto, always or never")
	depth := fs.Int("depth", 3, "maximum tree levels to show; 0 = unlimited")
	files := fs.Bool("files", false, "show individual files")
	sortBy := fs.String("sort", "churn", "sort siblings by churn or name")
	usageError := func(format string, a ...any) int {
		fmt.Fprintf(stderr, "blast-radius: "+format+"\n", a...)
		fs.Usage()
		return 2
	}
	// Everything after "--" is a pathspec. Before it, flags may come before
	// or after the revision, so parse flags, take a positional, and repeat.
	var pathspecs []string
	if i := slices.Index(args, "--"); i >= 0 {
		args, pathspecs = args[:i], args[i+1:]
	}
	var revs []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return 0
			}
			return 2
		}
		if fs.NArg() == 0 {
			break
		}
		revs, args = append(revs, fs.Arg(0)), fs.Args()[1:]
	}
	if len(revs) > 1 {
		return usageError("too many revisions: %s", strings.Join(revs, " "))
	}
	if *depth < 0 {
		return usageError("invalid --depth %d: must be 0 or more", *depth)
	}
	if *sortBy != "churn" && *sortBy != "name" {
		return usageError("invalid --sort %q: must be churn or name", *sortBy)
	}
	if *color != "auto" && *color != "always" && *color != "never" {
		return usageError("invalid --color %q: must be auto, always or never", *color)
	}

	rev := ""
	if len(revs) == 1 {
		rev = revs[0]
	}
	diff, err := gitdiff.Measure(gitdiff.Git{Env: env}, rev, pathspecs)
	if notRev := (gitdiff.NotRevisionError{}); errors.As(err, &notRev) {
		return usageError("%v", err)
	}
	if err != nil {
		fmt.Fprintf(stderr, "blast-radius: %v\n", err)
		return 1
	}
	for _, w := range diff.Warnings {
		fmt.Fprintf(stderr, "blast-radius: warning: %s\n", w)
	}
	label := diff.Label
	if diff.Range {
		label += " · " + render.Plural(diff.Commits, "commit")
	}
	root := tree.Build(diff.Files)
	tree.Shape(root, tree.Options{Files: *files, ByName: *sortBy == "name", Depth: *depth})
	useColor := render.UseColor(*color, isTerminal(stdout), getenv(env, "NO_COLOR"))
	io.WriteString(stdout, render.Render(label, diff.Subject, root, termWidth(stdout, env), useColor))
	return 0
}

// termWidth returns the terminal width: from stdout if it is a terminal,
// else from $COLUMNS, else 80.
func termWidth(stdout io.Writer, env []string) int {
	if isTerminal(stdout) {
		if w, _, err := term.GetSize(int(stdout.(*os.File).Fd())); err == nil && w > 0 {
			return w
		}
	}
	if w, err := strconv.Atoi(getenv(env, "COLUMNS")); err == nil && w > 0 {
		return w
	}
	return 80
}

// isTerminal reports whether w is a terminal.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// getenv returns the last value of key in env, or "".
func getenv(env []string, key string) string {
	v := ""
	for _, kv := range env {
		if k, val, ok := strings.Cut(kv, "="); ok && k == key {
			v = val
		}
	}
	return v
}
