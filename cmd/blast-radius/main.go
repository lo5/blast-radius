// Command blast-radius shows how many lines changed in each directory of a
// git repository.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/lo5/blast-radius/internal/gitdiff"
	"github.com/lo5/blast-radius/internal/render"
	"github.com/lo5/blast-radius/internal/tree"
)

// usage is the short usage, printed after usage errors.
const usage = `usage: blast-radius [flags] [<revision> | <A>..<B> | <A>...<B>] [-- <pathspec>...]
Run 'blast-radius --help' for more information.
`

// help is the full usage, printed by -h and --help.
const help = `usage: blast-radius [flags] [<revision> | <A>..<B> | <A>...<B>] [-- <pathspec>...]

Show how many lines changed in each directory of a git repository.

  blast-radius                 uncommitted changes (staged, unstaged, untracked) vs HEAD
  blast-radius <revision>      one commit vs its first parent
  blast-radius <A>..<B>        diff between the two endpoints
  blast-radius <A>...<B>       diff from merge-base(A, B) to B
  blast-radius ... -- <path>   only the given pathspecs, relative to the current directory

Flags:
  --depth N                    maximum tree levels to show; 0 = unlimited (default 3)
  --files                      show individual files
  --sort churn|name            sort siblings by churn or name (default churn)
  --exclude GLOB               exclude paths matching GLOB; repeatable
  --color auto|always|never    use color (default auto)
  --watch[=DUR]                redraw every DUR until interrupted (default 5s);
                               write --watch=1m, not --watch 1m
  --version                    print the version and exit
  -h, --help                   print this help and exit

Exclude globs are relative to the current directory unless they start with
':/' (relative to the repository root). '*' does not cross '/': use '**/*.md'
for Markdown files at any depth.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Environ()))
}

// run is the whole CLI: it parses args, collects all data, then writes the
// output in one go and returns the exit code. env is the environment for git
// and for settings such as COLUMNS.
func run(args []string, stdout, stderr io.Writer, env []string) int {
	fs := flag.NewFlagSet("blast-radius", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // usageError reports parse errors
	color := fs.String("color", "auto", "use color: auto, always or never")
	depth := fs.Int("depth", 3, "maximum tree levels to show; 0 = unlimited")
	files := fs.Bool("files", false, "show individual files")
	sortBy := fs.String("sort", "churn", "sort siblings by churn or name")
	showHelp := fs.Bool("help", false, "print help and exit")
	fs.BoolVar(showHelp, "h", false, "print help and exit")
	showVersion := fs.Bool("version", false, "print the version and exit")
	var every optString
	fs.Var(&every, "watch", "redraw every DUR until interrupted")
	var excludes []string
	fs.Func("exclude", "exclude paths matching a glob; repeatable", func(glob string) error {
		excludes = append(excludes, glob)
		return nil
	})
	usageError := func(format string, a ...any) int {
		fmt.Fprintf(stderr, "blast-radius: "+format+"\n", a...)
		io.WriteString(stderr, usage)
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
			return usageError("%v", err)
		}
		if fs.NArg() == 0 {
			break
		}
		revs, args = append(revs, fs.Arg(0)), fs.Args()[1:]
	}
	if *showHelp {
		io.WriteString(stdout, help)
		return 0
	}
	if *showVersion {
		info, ok := debug.ReadBuildInfo()
		fmt.Fprintf(stdout, "blast-radius %s — https://game-over.org\n", version(info, ok))
		return 0
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
	if every == "true" {
		every = "5s"
	}
	interval, err := time.ParseDuration(string(every))
	if every != "" && (err != nil || interval <= 0) {
		return usageError("invalid --watch=%s: must be a duration above 0, such as 5s", every)
	}
	if every != "" && !isTerminal(stdout) {
		return usageError("--watch needs a terminal")
	}

	rev := ""
	if len(revs) == 1 {
		rev = revs[0]
	}
	useColor := render.UseColor(*color, isTerminal(stdout), getenv(env, "NO_COLOR"))
	// frame measures and renders once, writing warnings to stderr.
	frame := func() (string, error) {
		diff, err := gitdiff.Measure(gitdiff.Git{Env: env}, rev, gitdiff.Pathspecs(pathspecs, excludes))
		if err != nil {
			return "", err
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
		suffix := filter(pathspecs, excludes)
		if every != "" {
			if suffix != "" {
				suffix += " · "
			}
			suffix += "every " + string(every) + " · " + time.Now().Format("15:04:05")
		}
		return render.Render(label, diff.Subject, suffix, root, termWidth(stdout, env), useColor), nil
	}
	out, err := frame()
	if notRev := (gitdiff.NotRevisionError{}); errors.As(err, &notRev) {
		return usageError("%v", err)
	}
	if err != nil {
		fmt.Fprintf(stderr, "blast-radius: %v\n", err)
		return 1
	}
	if every == "" {
		io.WriteString(stdout, out)
		return 0
	}
	// Watch until interrupted. Each frame clears the screen, which also
	// wipes any warnings, and later errors are shown in place of the tree.
	for {
		io.WriteString(stdout, "\x1b[H\x1b[2J"+out)
		time.Sleep(interval)
		if out, err = frame(); err != nil {
			out = fmt.Sprintf("blast-radius: %v\n", err)
		}
	}
}

// optString is a string flag that may also be given bare, like a bool flag,
// in which case its value is "true".
type optString string

func (s *optString) String() string     { return string(*s) }
func (s *optString) Set(v string) error { *s = optString(v); return nil }
func (s *optString) IsBoolFlag() bool   { return true }

// version returns the module version from the build info, or "devel" when
// there is none, as in a test binary or a build outside module mode.
func version(info *debug.BuildInfo, ok bool) string {
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "devel"
	}
	return info.Main.Version
}

// filter returns the excludes and pathspecs as they would be typed, such as
// `--exclude *.md -- src/`, or "" when the output is not filtered.
func filter(pathspecs, excludes []string) string {
	var parts []string
	for _, glob := range excludes {
		parts = append(parts, "--exclude "+glob)
	}
	if len(pathspecs) > 0 {
		parts = append(append(parts, "--"), pathspecs...)
	}
	return strings.Join(parts, " ")
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
