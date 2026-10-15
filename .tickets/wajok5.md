---
id: wajok5
status: closed
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

## Note 2026-10-15T18:21:09Z

From wuzan6: Git.Untracked substitutes ':/' only when pathspecs is empty. With excludes but no positive pathspec, ls-files would again be limited to the cwd while git diff covers the whole repo -- prepend ':/' whenever there is no positive pathspec (e.g. when only excludes were given).

## Note 2026-10-15T18:31:43Z

Done. gitdiff.Pathspecs(paths, excludes) appends ':(exclude,glob)GLOB' per exclude; a glob starting with ':/' becomes ':(exclude,glob,top)<rest>' (':(exclude,glob):/x' does not work in git). main passes Pathspecs(...) to Measure, so every mode gets the same list for diff and ls-files. Untracked prepends ':/' whenever there is no positive pathspec (isExclude recognizes ':(...exclude...)', ':!', ':^'). --exclude is fs.Func (repeatable) -> shows as '-exclude value' in the default usage; lazim7 should document it in --help incl. that globs are cwd-relative unless ':/' and that '*' doesn't cross '/' (use '**/*.md').
- Header: render.Render/Header gained a filter string param (after subject); main.filter() builds '--exclude G ... -- p1 p2' (excludes first, so it is a valid command line); suffix ' · <filter>' is never trimmed, the subject trims to make room. Spec §5.1, §5.4, §7.2 updated.
- Goldens: pathspec, exclude, range-filtered, subdirectory (TestSubdirectory compares src/api vs root for uncommitted, ':/' exclude, commit, range); bad-revision regenerated (usage gained -exclude).