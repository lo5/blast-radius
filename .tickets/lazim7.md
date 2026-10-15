---
id: lazim7
status: closed
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

## Note 2026-10-15T18:34:55Z

Done. main.go: const usage (synopsis + "Run 'blast-radius --help' for more information.") is printed by usageError for every exit-2 error; fs output is io.Discard and flag-package parse errors (unknown flag, bad int) also go through usageError, so they get the 'blast-radius: ' prefix. const help (full text, stdout, exit 0) lists invocation forms, flags (--depth N, --files, --sort churn|name, --exclude GLOB, --color auto|always|never, --version, -h/--help) and the exclude-glob note (cwd-relative unless ':/', '*' doesn't cross '/', use '**/*.md'). -h/-help/--help are Bool flags (not flag.ErrHelp), checked after the parse loop so 'foo -h' prints help; help wins over --version. version(info, ok) returns Main.Version or 'devel' for ''/'(devel)'; go build in a checkout gives a pseudo-version like v0.0.0-20261015183158-8291da149308+dirty. Goldens: help, version, unknown-flag; bad-revision regenerated. For the README (nukog5): testdata/help.golden is the canonical help text to copy; spec §4 and §8 updated.