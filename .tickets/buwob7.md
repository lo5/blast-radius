---
id: buwob7
status: closed
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

## Note 2026-10-15T18:28:06Z

Done. gitdiff.Range(g, arg, pathspecs) (Measure dispatches KindRange/KindMergeBase to it): RepoRoot first (exit 1 outside a repo), split arg at first "..."/"..", empty side -> HEAD, each side checked with rev-parse --verify --quiet <side>^{commit} -> NotRevisionError{arg} (exit 2; spec §5.3 clarified). git diff gets arg unchanged. Count: rev-list --count arg for A..B; for A...B, merge-base a b then rev-list --count <mb>..b. Diff gained Range bool + Commits int; main.go appends " · N commit(s)" (render.Plural) to the label before render.Render, so render's signature is unchanged and the subject-trim logic is untouched (ranges have no subject). wajok5: the filter suffix goes after the rest of the header; label for ranges already includes the count.
- Tests: gitdiff TestRange (unit, own small repo; shared testRepo(t) helper now used by TestCommit too); e2e goldens range (edit..feature, 2 commits) and range-merge-base (feature...edit, 1 commit, where the symmetric difference would be 3); TestRevisionArgs covers HEAD~1.., ...edit, root..nope, ../src, root^{tree}..HEAD.