---
id: obere4
status: open
type: feature
effort: normal
blocked-by: [fawen1]
created: 2026-10-15T18:04:21Z
---
# Proportional bars and width fitting

Each row gets a bar sized in eighths of a cell on one scale shared across the whole tree, so any two bars are comparable and a child is never longer than its parent. The row fits the terminal width.

Spec: §7.1, §7.3 column 2, §7.4.

- [ ] Width from `term.GetSize(stdout)` when a TTY, else `$COLUMNS`, else 80
- [ ] Bar width = remaining width, clamped to [10, 40]
- [ ] Length in eighths = round(churn / maxChurn × W × 8), min 1 when churn > 0; `█` cells then one of `▏▎▍▌▋▊▉`; maxChurn is over printed rows only
- [ ] Name column capped at 40% of width when it would overflow; names truncated from the end with `…`, tree prefix kept
- [ ] Unit tests: min 1 eighth, full width, truncation; all goldens regenerated and match the §7.1 layout