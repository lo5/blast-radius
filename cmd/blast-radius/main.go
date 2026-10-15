// Command blast-radius shows how many lines changed in each directory of a
// git repository.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

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
	fs.String("color", "auto", "use color: auto, always or never")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "blast-radius: unexpected argument %q\n", fs.Arg(0))
		fs.Usage()
		return 2
	}

	diff, err := gitdiff.Uncommitted(gitdiff.Git{Env: env}, nil)
	if err != nil {
		fmt.Fprintf(stderr, "blast-radius: %v\n", err)
		return 1
	}
	io.WriteString(stdout, render.Render(diff.Label, tree.Build(diff.Files)))
	return 0
}
