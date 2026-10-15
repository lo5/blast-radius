---
id: lazim7
status: open
type: feature
effort: easy
blocked-by: [kudag7]
created: 2026-10-15T18:04:21Z
---
# Help and version

`blast-radius -h/--help` prints usage with all flags; `--version` prints `blast-radius <version> — https://game-over.org`.

Spec: §4 flags, §5.4 (document glob relativity), §8 version note.

- [ ] Usage lists invocation forms and flags, and notes that exclude globs are relative to the current directory unless they start with `:/`
- [ ] Version from `runtime/debug.ReadBuildInfo()` module version, else `devel`
- [ ] Help and version exit 0; short usage reused by exit-2 errors