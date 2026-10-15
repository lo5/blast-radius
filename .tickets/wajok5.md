---
id: wajok5
status: open
type: feature
effort: normal
blocked-by: [kudag7]
created: 2026-10-15T18:04:21Z
---
# Pathspecs and excludes

`blast-radius [rev] -- src/` limits output to matching paths and `--exclude GLOB` (repeatable) removes matching paths, for every mode. The header shows the filter so readers know it is filtered. Output is identical regardless of which subdirectory it runs from.

Spec: §4, §5.4, §7.2 filter suffix.

- [ ] Pathspecs after `--` go to git unchanged (relative to cwd); each exclude becomes `:(exclude,glob)GLOB` after them, for both `git diff` and `git ls-files`
- [ ] No glob code in the tool
- [ ] Header appends ` · -- <pathspecs/excludes>` as typed
- [ ] Goldens: `-- src/`, `--exclude`, run from a subdirectory equals run from the root