---
id: kudag7
status: open
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