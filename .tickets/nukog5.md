---
id: nukog5
status: closed
type: chore
effort: easy
blocked-by: [hicid2, owuba6, buwob7, wajok5, lazim7]
created: 2026-10-15T18:04:21Z
---
# README

A README that lets a new user install and use the tool without reading the spec.

- [ ] `go install` line, requirements (Go, git >= 2.28), supported platforms
- [ ] Usage table and flags, sample output matching the current goldens
- [ ] Notes on simplifications: no rename detection, binary files skipped, no default excludes

## Note 2026-10-15T18:36:01Z

README.md written: intro with sample output copied from testdata/shape-default.golden, install/requirements/platforms, usage + invocation table, flags table (wording from help.golden), exclude-glob note with the range-filtered.golden example (args match TestRangeFiltered), simplifications. Help text verified against built binary. No spec changes needed. If help or goldens change, update README samples.