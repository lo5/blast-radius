// Command blast-radius shows how many lines changed in each directory of a
// git repository.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
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
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	usageError := func(format string, a ...any) int {
		fmt.Fprintf(stderr, "blast-radius: "+format+"\n", a...)
		fs.Usage()
		return 2
	}
	if fs.NArg() > 0 {
		return usageError("unexpected argument %q", fs.Arg(0))
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

	diff, err := gitdiff.Uncommitted(gitdiff.Git{Env: env}, nil)
	if err != nil {
		fmt.Fprintf(stderr, "blast-radius: %v\n", err)
		return 1
	}
	for _, w := range diff.Warnings {
		fmt.Fprintf(stderr, "blast-radius: warning: %s\n", w)
	}
	root := tree.Build(diff.Files)
	tree.Shape(root, tree.Options{Files: *files, ByName: *sortBy == "name", Depth: *depth})
	useColor := render.UseColor(*color, isTerminal(stdout), getenv(env, "NO_COLOR"))
	io.WriteString(stdout, render.Render(diff.Label, root, termWidth(stdout, env), useColor))
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
