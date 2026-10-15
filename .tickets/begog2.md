---
id: begog2
status: closed
type: feature
effort: normal
children:
  - fawen1
  - hicid2
  - obere4
  - owuba6
  - wuzan6
  - kudag7
  - buwob7
  - wajok5
  - lazim7
  - nukog5
created: 2026-10-15T18:04:20Z
---
# blast-radius v1

Implement the blast-radius CLI as described in docs/spec.md: a directory tree of changed lines with proportional bars, +/- counts, file counts and % of total, for uncommitted changes, a single commit, or a commit range. Stdlib plus golang.org/x/term only; git does the work.