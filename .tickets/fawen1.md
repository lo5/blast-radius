---
id: fawen1
status: closed
type: feature
effort: hard
created: 2026-10-15T18:04:20Z
---
# Walking skeleton: uncommitted tracked changes

Running `blast-radius` with no arguments in a repo that has staged and unstaged edits prints the header (`working tree vs HEAD · N files · +A -D`), a blank line, then a tree of directory/file rows with name, `+added -deleted`, file count and percent. No bars yet; no file hiding, collapsing, sorting or depth limit yet (rows in a stable order).

Spec: §1, §5 (common diff flags, repo root, empty-tree hash), §5.1 step 1, §5.5, §6 (build + aggregate), §7.2, §7.3 (minus bar column), §8, §9, §10.

- [ ] Module `github.com/lo5/blast-radius`, `go 1.27`, binary in `cmd/blast-radius`, package layout per §9
- [ ] In-process entry point `run(args, stdout, stderr, env) int`; `main` only wires it to `os.Exit`
- [ ] git runner captures stdout/stderr; every diff uses `--numstat -z --no-renames --no-ext-diff --no-textconv --no-color --no-relative`
- [ ] numstat parser drops binary (`-`) and 0/0 records; handles paths with spaces, tabs, newlines, non-ASCII (unit tests)
- [ ] Tree build aggregates Added/Deleted/Files up to the root; root is never a row (unit tests)
- [ ] Rows use `├─ `/`└─ `/`│  `/`   ` prefixes, dirs end with `/`, columns aligned per §7.3, thousands separators, `1 file`/`N files`, `<1%`
- [ ] Nothing to show: header then `no changes`, exit 0
- [ ] Not a repo / git failure: `blast-radius: <git stderr>`, exit 1; git missing: `blast-radius: git not found in PATH`, exit 1
- [ ] All data is collected before anything is written to stdout
- [ ] Golden harness: throwaway repos in `t.TempDir()`, fixed git env (`GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_NOSYSTEM=1`, fixed author/committer/dates), `--color=never` behavior, `COLUMNS=100`, `go test ./... -update` rewrites `testdata/*.golden`
- [ ] Goldens: uncommitted staged+unstaged, no changes, not a git repo

## Note 2026-10-15T18:11:35Z

Done. Layout: cmd/blast-radius/main.go (run(args, stdout, stderr, env []string) int; flag.FlagSet, ContinueOnError, output to stderr; any positional arg -> exit 2 for now), internal/gitdiff/{git.go,numstat.go,target.go}, internal/tree/tree.go, internal/render/render.go. go.mod has no deps yet (add golang.org/x/term with the bars ticket).
- FileStat lives in package tree; gitdiff aliases it (gitdiff.FileStat = tree.FileStat).
- gitdiff.Git{Env}.Run(args...) returns stdout; errors are git's trimmed stderr, or "git not found in PATH". Git.RepoRoot() and Git.EmptyTree() exist (EmptyTree unused so far: wuzan6/kudag7). Git.diff(revs, pathspecs) adds the common diff flags and "--". gitdiff.Uncommitted(g, pathspecs) returns Diff{Label, Files}; unborn HEAD currently fails with git's "ambiguous argument 'HEAD'" (wuzan6).
- tree.Build keeps children in first-appearance order (git's path order); no sorting/hiding/collapsing yet (hicid2). Files are shown as leaves everywhere for now.
- render.Render(label, root) returns the whole output; exported helpers Thousands, Plural, Percent, Header. Widths use rune count. No bar column yet (obere4: insert between name and counts).
- --color is registered but ignored and unvalidated (owuba6).
- Harness (cmd/blast-radius/main_test.go): testEnv(t), newRepo(t), git(t, dir, args...), write(t, dir, path, content), lines(prefix, n), commitAll(t, dir, msg), runCLI(t, dir, args...) (t.Chdir + --color=never; appends "-- stderr --"/"-- exit N --" when notable), golden(t, name, got). testdata path is made absolute at init because of t.Chdir. Every test package declares a dummy -update flag so `go test ./... -update` works; new test packages must too.
- Spec: §8 now says header, blank line, then `no changes`; §10 documents golden format.