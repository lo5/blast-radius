---
id: wuzan6
status: open
type: feature
effort: normal
blocked-by: [fawen1]
created: 2026-10-15T18:04:21Z
---
# Untracked files and unborn HEAD

Uncommitted mode also counts new untracked text files (all lines added), and works in a repo with no commits yet.

Spec: §5.1 steps 2–3, §8 unreadable-file row.

- [ ] Untracked list from `git ls-files --others --exclude-standard --full-name -z`, with the user's pathspecs passed through
- [ ] Skip entries ending in `/` (nested repos) and anything that isn't a regular file per `os.Lstat` (symlinks, sockets)
- [ ] Skip binary files (NUL in first 8 KiB)
- [ ] Line count streams the file: number of `\n`, plus 1 if non-empty without trailing `\n`; empty files dropped
- [ ] Unreadable file: warning on stderr, skip, exit 0
- [ ] Unborn HEAD diffs against the empty tree (hash computed, not hard-coded)
- [ ] Unit tests: trailing newline / none, empty, binary, symlink; goldens: uncommitted with staged+unstaged+untracked, repo with no commits, binary files ignored