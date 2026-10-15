---
id: owuba6
status: closed
type: feature
effort: normal
blocked-by: [obere4]
created: 2026-10-15T18:04:21Z
---
# Color output

On a terminal, the added part of each bar and the `+N` count are green and the deleted part and `-N` count are red; everything else stays default. Piped output and `NO_COLOR` stay plain.

Spec: §4 `--color`, §7.5.

- [ ] `--color auto|always|never`; auto = stdout is a TTY and `NO_COLOR` unset or empty; `always` ignores `NO_COLOR`
- [ ] Only SGR 32 (green) and 31 (red)
- [ ] Bar split: of n drawn cells (partial counts as one), the first round(n × added / churn) are green, the rest red
- [ ] Bad `--color` value: exit 2 with usage
- [ ] Unit tests for split rounding; one golden with `--color=always`

## Note 2026-10-15T18:18:56Z

Done. internal/render/color.go: UseColor(mode, tty, noColor) (pure; main passes isTerminal(stdout) and getenv(env, "NO_COLOR")), GreenCells(n, added, churn) = round(n*added/churn), colorBar(bar, added, churn), paint(code, s) wraps non-empty s in ESC[<code>m ... ESC[0m.
- render.Render(label, root, termWidth, color bool): padding is computed from the plain strings (new pad(s, w) helper), then the bar and +/- counts are colored. Counts are colored even when zero.
- main.go: --color validated with the other flags (usageError, exit 2); new isTerminal(w) helper shared by termWidth and color auto.
- Tests: render unit tests for GreenCells rounding, colorBar, UseColor, and Render(color) == plain once SGR is stripped. e2e: TestColorAlways (own mixed add/delete fixture, NO_COLOR=1 set to prove always wins) -> testdata/color-always.golden; TestColorAutoPiped (no flag, bytes.Buffer stdout -> no ESC). runCLI still prepends --color=never; a later --color=always overrides it.
- Spec §7.5: said that zero counts are colored too, runs end with SGR 0, and padding counts cells.