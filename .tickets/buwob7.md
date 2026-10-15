---
id: buwob7
status: open
type: feature
effort: normal
blocked-by: [kudag7]
created: 2026-10-15T18:04:21Z
---
# Commit ranges

`blast-radius A..B` and `A...B` show the diff between endpoints (or from the merge-base), with the commit count in the header: `main..feature · 12 commits · ...`.

Spec: §5.3, §7.2.

- [ ] Any argument containing `..` is passed to `git diff` unchanged; empty sides default to HEAD as in git
- [ ] `A..B` count from `rev-list --count A..B`; `A...B` count from `rev-list --count $(merge-base A B)..B` (not the symmetric difference)
- [ ] Label is the range as typed; `1 commit` singular
- [ ] Goldens: `A..B` and `A...B` with correct counts