---
id: kudag7
status: closed
type: feature
effort: normal
blocked-by: [fawen1]
created: 2026-10-15T18:04:21Z
---
# Single commit revision argument

`blast-radius <rev>` shows one commit against its first parent (or the empty tree for a root commit), with the header `<short sha> <subject>`.

Spec: §4 usage rules, §5.2, §8 exit-2 rows.

- [ ] Arg parsing loop: flags may come before or after the one positional, not after `--`; everything after `--` is kept as pathspecs
- [ ] More than one revision: usage error, exit 2
- [ ] Non-revision argument: `blast-radius: 'foo' is not a revision (use '-- foo' to filter by path)`, exit 2
- [ ] Resolve with `rev-parse --verify --quiet <arg>^{commit}`; parents from `rev-list --parents -n 1`; diff against `^1` or empty tree
- [ ] Header from `git log -1 --format='%h %s'`, subject trimmed to fit the terminal width
- [ ] Unit tests: argument classification (none / revision / `..` / `...`), header labels; goldens: single commit, root commit, merge commit (first parent), bad revision, moved file = delete + add

## Note 2026-10-15T18:25:29Z

Done. main.go: args split at the first literal "--" (slices.Index) before flag parsing -> pathspecs; then a loop of fs.Parse + take fs.Arg(0) collects revs, so flags work before/after the revision. >1 rev: usageError "too many revisions: A B", exit 2. Pathspecs are already passed to gitdiff.Measure (and so to git diff / ls-files), but the header filter suffix and excludes are NOT done (wajok5).
- gitdiff/target.go: Kind consts KindUncommitted/KindCommit/KindRange/KindMergeBase and Classify(arg) ("..." checked before ".."); Measure(g, arg, pathspecs) dispatches; ranges currently return errors.New("commit ranges are not supported yet") (exit 1) -- buwob7 replaces that case. NotRevisionError{Arg}; main maps it via errors.As to usageError (message + usage, exit 2). Commit() calls RepoRoot first so "not a git repository" stays exit 1; rev-parse --verify --quiet <rev>^{commit}; base = first parent from rev-list --parents (fields[1]) or EmptyTree; label/subject from log -1 --format=%h %s split at the first space.
- Diff gained Subject. render.Render(label, subject, root, width, color) and Header(label, subject, root, width): only the subject is trimmed with … (via truncate) so the header fits; label and the " · N files · +a -d" rest are never trimmed. buwob7 can put the commit count into the header the same way (new param or field); wajok5 appends the filter suffix -- the subject trim must account for both.
- e2e: historyRepo(t) fixture (tags root, edit; feature branch merged --no-ff; final commit moves src/core -> lib). Goldens commit (long subject trimmed at COLUMNS=100, flags after rev), commit-root, commit-merge, commit-move, bad-revision (includes flag default usage -> lazim7 will need -update). TestRevisionArgs/TestRevisionNotARepo assert without goldens.
- Spec: §4 says args split at first "--" (flag value "--" needs --flag=--); §5.2 clarifies subject trimming, not-a-repo precedence, non-commit objects are non-revisions.