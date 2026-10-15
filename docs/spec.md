# blast-radius — Specification

Status: draft, ready for implementation
Module: `github.com/lo5/blast-radius`
Binary: `blast-radius`

## 1. Overview

`blast-radius` is a Go command-line tool that shows, at a glance, how many lines
changed in each directory of a git repository. It measures one of:

- **uncommitted changes** (staged + unstaged + untracked, compared to `HEAD`),
- **a single commit** (compared to its first parent), or
- **a range of commits** (`A..B` or `A...B`).

It prints a directory tree with one proportional bar per row, plus line counts,
file counts and share of total.

### Design principle: keep it simple

Accurate-enough numbers from simple logic beat precise numbers from complex
logic. When there's a choice, pick the option with less code and let git do the
work. That is why the tool skips rename detection, drops binary files, and
hands exclude globs to git's own pathspec matching.

## 2. Goals and non-goals

**Goals**

- A single command that gives a compact, readable picture of where a change landed.
- Revision arguments that work like git's, so there's nothing new to learn.
- Output that looks the same no matter which directory inside the repo you run it from.
- Few dependencies, so the code is easy to read and maintain.

**Non-goals (v1)**

- JSON, HTML, or other output formats.
- An interactive or full-screen TUI.
- Rename/move detection. A move counts as a delete plus an add.
- Measuring binary files.
- Default excludes such as vendor/, lockfiles, or generated code.
- Per-commit breakdowns, authorship/ownership, or blame.
- Release pipelines or prebuilt binaries. Users install with `go install`.

## 3. Installation and requirements

```sh
go install github.com/lo5/blast-radius/cmd/blast-radius@latest
```

- Go toolchain: `go 1.27` in `go.mod`.
- Runtime: `git` ≥ 2.28 on `PATH`. 2.28 is needed for `--no-relative`; the
  `:(exclude,glob)` pathspec has been available since 2.13.
- Platforms: macOS and Linux are supported and tested. Windows is best-effort.
- Dependencies: the standard library plus `golang.org/x/term` for TTY and width
  detection. Nothing else.

## 4. Usage

```
blast-radius [flags] [<revision> | <A>..<B> | <A>...<B>] [-- <pathspec>...]
```

| Invocation                         | Measures                                                    |
|------------------------------------|-------------------------------------------------------------|
| `blast-radius`                     | Uncommitted changes: staged + unstaged + untracked vs `HEAD` |
| `blast-radius HEAD~1`              | That one commit vs its first parent                         |
| `blast-radius abc123`              | That one commit vs its first parent                         |
| `blast-radius main..feature`       | Diff between the two endpoints                              |
| `blast-radius v1.2...HEAD`         | Diff from `merge-base(v1.2, HEAD)` to `HEAD`                |
| `blast-radius HEAD~5.. -- src/`    | Range, limited to paths under `src/`                        |
| `blast-radius -- .`                | Uncommitted, limited to the current directory               |

Rules:

- At most one revision argument. More than one is a usage error (exit 2).
- Everything after `--` is passed to git unchanged as pathspecs. Git interprets
  them relative to the current directory, so `-- .` means "here".
- Flags may come before or after the revision argument, but not after `--`.
  Go's `flag` package stops at the first positional argument, so parse in a loop:
  parse flags, take one positional, then parse the rest.
- If the single argument is not a valid revision, fail with exit 2 and suggest
  `--`, the same way git handles ambiguous arguments:
  `blast-radius: 'foo' is not a revision (use '-- foo' to filter by path)`.

### Flags

| Flag                         | Default | Meaning                                                                 |
|------------------------------|---------|-------------------------------------------------------------------------|
| `--depth N`                  | `3`     | Maximum displayed tree levels. `0` = unlimited.                         |
| `--files`                    | off     | Show individual files as leaves. Off = directories only.                |
| `--sort churn\|name`         | `churn` | Sort order of siblings.                                                 |
| `--exclude GLOB`             | none    | Exclude matching paths. Repeatable.                                     |
| `--color auto\|always\|never`| `auto`  | Whether to use color.                                                   |
| `--version`                  |         | Print version and exit.                                                 |
| `-h`, `--help`               |         | Print usage and exit.                                                   |

## 5. Choosing what to measure

All git commands run from the current directory. Every diff command includes
these flags so that user git config and diff drivers can't change the output:

```
--numstat -z --no-renames --no-ext-diff --no-textconv --no-color --no-relative
```

`--no-relative` makes paths relative to the repo root even when the tool runs in
a subdirectory. Repo root comes from `git rev-parse --show-toplevel`.

The empty tree is computed with `git hash-object -t tree --stdin </dev/null`
rather than hard-coded, so SHA-256 repositories work too.

### 5.1 Uncommitted (no revision argument)

1. Tracked changes: `git diff HEAD <common flags> -- <pathspecs>`.
   This covers staged and unstaged changes in one diff.
2. Untracked files:
   `git ls-files --others --exclude-standard --full-name -z -- <pathspecs>`.
   - Skip entries ending in `/` (nested repositories).
   - `os.Lstat` each file relative to the repo root. Skip anything that isn't a
     regular file, such as symlinks or sockets.
   - Treat the file as binary, and skip it, if the first 8 KiB contains a NUL byte.
   - Otherwise count lines by streaming the file: lines = number of `\n`, plus 1
     if the file is non-empty and doesn't end in `\n`. All lines count as added.
3. Unborn `HEAD` (a repo with no commits yet): diff against the empty tree
   instead of `HEAD`.

Header label: `working tree vs HEAD`.

### 5.2 Single commit

1. Resolve the argument with `git rev-parse --verify --quiet <arg>^{commit}`.
2. Find its parents with `git rev-list --parents -n 1 <sha>`.
3. If it has parents: `git diff <sha>^1 <sha> <common flags> -- <pathspecs>`.
   This uses the first parent, so a merge commit shows everything it brought
   into the mainline.
4. If it's a root commit: `git diff <emptytree> <sha> ...`.

Header label: `<short sha> <subject>` from `git log -1 --format='%h %s'`. Trim
the subject so the header fits the terminal width.

### 5.3 Ranges

- Any argument containing `..` is a range. Pass it straight to `git diff`.
  `A..B` compares the two endpoints; `A...B` compares `merge-base(A,B)` to `B`.
  An empty side defaults to `HEAD`, as in git.
- Commit count for the header:
  - `A..B`: `git rev-list --count A..B`.
  - `A...B`: `git rev-list --count $(git merge-base A B)..B`. Don't use
    `rev-list A...B`, which counts the symmetric difference.

Header label: the range as the user typed it, e.g. `main..feature · 12 commits`.

### 5.4 Excludes

Each `--exclude GLOB` becomes the pathspec `:(exclude,glob)GLOB` and is added
after the user's pathspecs, for both `git diff` and `git ls-files`. Git does all
the glob matching, so the tool contains no glob code. Like git, globs are
relative to the current directory unless they start with `:/`. Document this in
`--help`.

### 5.5 Parsing numstat

With `-z --no-renames`, each record is `<added>\t<deleted>\t<path>\0`.

- If either count is `-`, the file is binary: drop the record.
- Drop records where both counts are `0`, such as mode-only changes or new empty
  files. Only changed lines count.
- Submodule pointer changes appear as `1\t1\t<path>`. Keep them as they are;
  no special handling.

## 6. Data model and aggregation

```go
type FileStat struct {
    Path    string // slash-separated, relative to repo root
    Added   int
    Deleted int
}

type Node struct {
    Name     string  // display label; may hold a collapsed chain like "internal/util"
    IsDir    bool
    Added    int     // sum over everything under this node
    Deleted  int
    Files    int     // count of changed files under this node
    Children []*Node
}
```

- `Churn = Added + Deleted`.
- Build a full tree from all `FileStat`s, splitting paths on `/`. Each directory
  node holds the totals for everything below it.
- The root node is the repo root. It is never printed as a row; its totals go in
  the header.

### 6.1 Steps after building the tree (in this order)

1. **Hide files.** Unless `--files` is set, remove file nodes from every
   directory except the root. Totals stay the same; only the leaves disappear.
   Files at the repo root are always shown.
2. **Collapse chains.** If a directory's only child is another directory and it
   has no directly visible files, merge the two. Repeat until nothing changes.
   The label joins the names: `internal/util/`. The merged row keeps the outer
   directory's totals, so files hidden in step 1 still count toward it. Only
   directories merge: with `--files`, `a/b/c.go` shows as `a/b/` with `c.go`
   below it. The root is never merged into its child.
3. **Sort.** At every level, sort by churn (highest first, ties by name ascending).
   With `--sort name`, sort by name ascending. Directories and files are mixed
   together.
4. **Limit depth.** Print rows only down to level `--depth`, counting the root's
   children as level 1. Rows below that are hidden, but their counts are already
   included in their ancestors. Depth counts levels after collapsing, so deep
   single-child paths like `src/main/java/com/acme/` take up only one level.

## 7. Rendering

### 7.1 Example (`--color=never`)

```
$ blast-radius main..feature
main..feature · 12 commits · 34 files · +1,204 -388

src/               ████████████████████████  +1,010 -300  26 files  82%
├─ api/            █████████████▉              +620 -140  14 files  48%
│  ├─ handlers/    ██████████▌                 +480  -90  10 files  36%
│  └─ middleware/  ███▌                        +140  -50   4 files  12%
├─ core/           ███████▊                    +300 -120   9 files  26%
└─ internal/util/  ██▍                          +90  -40   3 files   8%
docs/              ███▉                        +150  -60   5 files  13%
scripts/           █▎                           +40  -25   2 files   4%
go.mod             ▏                             +4   -3    1 file  <1%
```

### 7.2 Header

```
<label> · [<N> commits · ]<F> files · +<added> -<deleted>[ · -- <pathspecs/excludes>]
```

- Labels are defined in §5.1–5.3. The commit count appears only for ranges.
- If pathspecs or excludes are set, append them as typed so readers know the
  output is filtered.
- Print a blank line after the header.
- Format numbers with thousands separators, in the header and in rows.
- Use `file`/`commit` when the count is 1.

### 7.3 Row columns

Columns, separated by two spaces:

1. **Name**: tree prefix plus label. Directories end with `/`. Left-aligned and
   padded to the widest name. Prefixes are `├─ `, `└─ `, `│  `, and `   `.
   Top-level rows have no prefix.
2. **Bar**: left-aligned and padded to the bar width (§7.4).
3. **Counts**: `+added` and `-deleted`, each right-aligned to the widest value in
   its column, separated by one space. Use ASCII `-` so the output is easy to grep.
4. **Files**: `N files` or `1 file`, right-aligned.
5. **Percent**: `round(churn / total churn × 100)%`, right-aligned. A non-zero
   value that rounds below 1 shows as `<1%`.

### 7.4 Bars

- Terminal width comes from `term.GetSize(stdout)` when stdout is a TTY,
  otherwise from `$COLUMNS`, otherwise 80.
- Bar width W = terminal width − (all other columns + separators), limited to
  the range [10, 40].
- If the name column would push the row past the terminal width even with the
  minimum bar width, cap it at 40% of the width and shorten names from the end
  with `…`. Keep the tree prefix. On very narrow terminals the row can still be
  wider than the terminal, since the bar never drops below 10 cells.
- Scale: `maxChurn` is the largest churn among the rows that will be printed.
  Bar length in eighths of a cell = `round(churn / maxChurn × W × 8)`, at least 1
  when churn > 0. Draw full cells as `█`, followed by one partial cell from
  `▏▎▍▌▋▊▉` if needed.
- Because the scale is shared across the whole tree, any two bars can be
  compared, and a child's bar is never longer than its parent's.

### 7.5 Color

- `auto`: use color only if stdout is a TTY and `NO_COLOR` is unset or empty.
  `always` and `never` override this. `NO_COLOR` doesn't override `always`.
- Use only two colors. Green (SGR 32) for the added part of the bar and the
  `+N` count; red (SGR 31) for the deleted part and the `-N` count, including
  `+0` and `-0`. Each colored run ends with a reset (SGR 0). Everything else
  stays the default color. Padding counts cells, not escape codes.
- To split a bar into colors, let n be the number of cells drawn, counting a
  partial cell as one. The first `round(n × added / churn)` cells are green and
  the rest are red. The partial cell, if any, takes the color of whichever side
  it falls on.
- Without color, bars show as one block. The counts still show the +/- split.

## 8. Edge cases and exit codes

| Situation                                          | Behavior                                                       | Exit |
|----------------------------------------------------|----------------------------------------------------------------|------|
| Nothing to show after filtering                    | Print the header line, a blank line, then `no changes`         | 0    |
| Success                                            | Header and tree                                                | 0    |
| `git` not on `PATH`                                | `blast-radius: git not found in PATH`                          | 1    |
| Not inside a git repo / git command fails          | `blast-radius: ` + git's stderr (trimmed)                      | 1    |
| Unreadable untracked file                          | Warning on stderr, skip the file, continue                     | 0    |
| Bad flag value, >1 revision, non-revision argument | Message plus short usage on stderr                             | 2    |

- Never print partial output on stdout and then fail. Collect all data before
  rendering.
- `--version` prints `blast-radius <version> — https://game-over.org`. The
  version comes from `runtime/debug.ReadBuildInfo()` (the module version for
  `go install`, otherwise `devel`).

## 9. Project layout

```
blast-radius/
├── go.mod
├── README.md
├── cmd/blast-radius/
│   └── main.go          # flag parsing, wiring, exit codes
├── internal/gitdiff/
│   ├── git.go           # run git (exec.Command), capture stdout/stderr
│   ├── target.go        # pick target (uncommitted/commit/range), header label, commit count
│   ├── numstat.go       # parse -z numstat output
│   └── untracked.go     # list untracked files, detect binary, count lines
├── internal/tree/
│   └── tree.go          # build, aggregate, hide files, collapse, sort, limit depth
├── internal/render/
│   ├── render.go        # columns, header, number formatting
│   ├── bar.go           # bar length and color split
│   └── color.go         # decide color, SGR helpers
├── testdata/            # golden files for e2e tests
└── docs/spec.md
```

Keep packages small. `tree` and `render` contain only pure functions; all git
and filesystem I/O is in `gitdiff`.

## 10. Testing

**Unit tests**

- `numstat`: normal records, binary `-` records, zero-zero records, paths with
  spaces, tabs, newlines, and non-ASCII characters (the `-z` path).
- `untracked`: line counts with and without a trailing newline, empty files,
  binary files, symlinks skipped.
- `tree`: aggregation totals, hiding files while keeping root files, chain
  collapsing (including chains that end in a file when `--files` is set),
  depth limits after collapsing, both sort orders, tie-breaking.
- `render`: bar eighths at the boundaries (minimum 1 eighth, full width), color
  split rounding, percent formatting (`<1%`), thousands separators, name
  truncation, singular and plural wording.
- `target`: argument classification (none / revision / `..` / `...`), the
  non-revision error, header labels.

**End-to-end golden tests**

- Helpers create throwaway repos in `t.TempDir()` with git run under a fixed
  environment: `GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_NOSYSTEM=1`, fixed
  author/committer names, emails and dates.
- Run the CLI's `run(args, stdout, stderr, env) int` entry point in-process,
  with `--color=never` and `COLUMNS=100` so output is deterministic. Compare
  against `testdata/*.golden`. `go test ./... -update` rewrites the golden files.
  A golden file holds stdout, followed by `-- stderr --`, stderr and
  `-- exit N --` when stderr is non-empty or the exit code is non-zero. Every
  test package accepts `-update` so the flag works with `./...`.
- Scenarios:
  - uncommitted with staged, unstaged, and untracked files
  - uncommitted in a repo with no commits yet
  - single commit
  - root commit
  - merge commit (first-parent)
  - `A..B` and `A...B` with the correct commit counts
  - pathspec `-- src/`
  - `--exclude`
  - `--files`
  - `--depth 1` and `--depth 0`
  - `--sort name`
  - running from a subdirectory gives the same output as from the root
  - binary files ignored
  - a moved file counts as delete plus add
  - no changes
  - not a git repo (exit 1)
  - bad revision (exit 2)
  - one golden file with `--color=always` to lock in the SGR output

## 11. Possible future work (out of scope for v1)

- `--json` output.
- Optional rename detection (`-M`).
- A "breadth" score, e.g. number of top-level directories touched.
- `git-blast-radius` symlink so it can run as `git blast-radius`.
- A vanity import path under game-over.org.
