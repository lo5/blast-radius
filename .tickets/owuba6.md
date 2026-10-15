---
id: owuba6
status: open
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