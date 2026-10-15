---
id: obere4
status: closed
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

## Note 2026-10-15T18:17:09Z

Done. render.Render(label, root, termWidth) now draws bars. internal/render/bar.go: Eighths(churn, maxChurn, w) (min 1 when churn>0) and Bar(eighths) returning unpadded █…+partial; min/maxBarWidth consts. Render: maxChurn = max over flattened (printed) rows; rest = counts+files+pct+4 separators; if nameW+rest+10 > width, nameW = min(nameW, width*2/5); barW = clamp(width-nameW-rest, 10, 40). Rows keep prefix and label separately; truncate(prefix, label, w) cuts the label with … and always keeps the prefix. Color ticket (owuba6): split the Bar cells there; Render pads with padRight so SGR codes will need width-aware padding (pad by cell count, not string width).
- main.go: termWidth(stdout, env) uses term.GetSize when stdout is an *os.File TTY, else COLUMNS from env (getenv(env, key), last wins, must be >0), else 80. Color 'auto' can reuse the same *os.File + term.IsTerminal check.
- go.mod requires golang.org/x/term v0.47.0 (x/sys v0.49.0 indirect); go.sum added.
- Harness: runCLIEnv(t, dir, env, args...) for custom env; golden narrow.golden (COLUMNS=50: names truncated, row still overflows since bar min is 10). All goldens regenerated with bars.
- Spec §7.4 clarified: cap applies when name + other columns + minimum bar overflow; rows may still overflow on very narrow terminals.