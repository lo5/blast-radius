# blast-radius

See at a glance how many lines changed in each directory of a git repo: for uncommitted
changes, a single commit, or a range of commits.

```
$ blast-radius
working tree vs HEAD · 11 files · +395 -1

src/                     ████████████████████████████████████████  +325 -0  6 files  82%
├─ api/                  ██████████████████████████████▊           +250 -0  4 files  63%
│  ├─ handlers/          ████████████████████████▋                 +200 -0  2 files  51%
│  └─ middleware/        ████▉                                      +40 -0   1 file  10%
├─ core/                 ███████▍                                   +60 -0   1 file  15%
└─ internal/util/        █▉                                         +15 -0   1 file   4%
java/src/main/com/acme/  ███▊                                       +30 -0   1 file   8%
docs/                    ███▏                                       +25 -0   1 file   6%
scripts/                 ▉                                           +7 -0   1 file   2%
tools/                   ▉                                           +7 -0   1 file   2%
go.mod                   ▎                                           +1 -1   1 file   1%
```

Each row shows a directory's bar (scaled to the largest row), lines added and deleted, changed
files, and its share of the total churn. Single-child directory chains collapse into one row
(`internal/util/`). In a terminal, bars and counts are green for additions and red for
deletions.

## Install

```sh
go install github.com/lo5/blast-radius/cmd/blast-radius@latest
```

From a checkout, `make build` writes `bin/blast-radius` and `make install` installs it into
`$(go env GOPATH)/bin`. `make test` runs the tests and `make clean` removes `bin/`.

Requires Go 1.27+ to build and `git` 2.28+ on `PATH`. Supported on macOS and Linux; Windows is
best-effort.

## Usage

```
blast-radius [flags] [<revision> | <A>..<B> | <A>...<B>] [-- <pathspec>...]
```

| Invocation                      | Measures                                                  |
|---------------------------------|-----------------------------------------------------------|
| `blast-radius`                  | Uncommitted changes (staged, unstaged, untracked) vs HEAD |
| `blast-radius HEAD~1`           | That one commit vs its first parent                       |
| `blast-radius main..feature`    | Diff between the two endpoints                            |
| `blast-radius main...feature`   | Diff from `merge-base(main, feature)` to `feature`        |
| `blast-radius HEAD~5.. -- src/` | Range, limited to paths under `src/`                      |
| `blast-radius -- .`             | Uncommitted, limited to the current directory             |

Revisions and ranges work as in git. Pathspecs after `--` go to git unchanged and are relative
to the current directory. Output is the same from any directory in the repo.

| Flag                          | Default | Meaning                                       |
|-------------------------------|---------|-----------------------------------------------|
| `--depth N`                   | `3`     | Maximum tree levels to show; `0` = unlimited  |
| `--files`                     | off     | Show individual files                         |
| `--sort churn\|name`          | `churn` | Sort siblings by churn or name                |
| `--exclude GLOB`              | none    | Exclude paths matching GLOB; repeatable       |
| `--color auto\|always\|never` | `auto`  | Use color (`auto` honors `NO_COLOR`)          |
| `--version`                   |         | Print the version and exit                    |
| `-h`, `--help`                |         | Print help and exit                           |

Exclude globs are relative to the current directory unless they start with `:/` (relative to
the repository root). `*` does not cross `/`: use `**/*.md` for Markdown files at any depth.
Filters are echoed in the header:

```
$ blast-radius edit..feature --exclude 'src/api/handlers/**' -- src/
edit..feature · 2 commits · 2 files · +25 -16 · --exclude src/api/handlers/** -- src/

src/      ████████████████████████████████████████  +25 -16  2 files  100%
├─ api/   ████████████████████████▍                 +15 -10   1 file   61%
└─ core/  ███████████████▋                          +10  -6   1 file   39%
```

## Simplifications

Numbers are accurate enough, not exact, by design:

- No rename detection: a moved file counts as a delete plus an add.
- Binary files are skipped.
- No default excludes: vendored code, lockfiles and generated files count unless you pass
  `--exclude`.
- A single commit is compared to its first parent, so a merge shows everything it brought in.
