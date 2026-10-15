---
id: hicid2
status: open
type: feature
effort: normal
blocked-by: [fawen1]
created: 2026-10-15T18:04:20Z
---
# Tree shaping: hide files, collapse chains, sort, depth

Output takes its final shape: directories only by default (root files always shown), single-child directory chains collapse into one row like `internal/util/`, siblings sort by churn (ties by name) or by name, and only `--depth` levels are printed.

Spec: §4 flags `--depth`, `--files`, `--sort`; §6.1.

- [ ] `--files` shows file leaves; default hides them except at the repo root; totals unchanged
- [ ] Chain collapsing repeats until stable, including chains ending in a file with `--files`
- [ ] `--sort churn` (default; desc, ties by name asc) and `--sort name`; dirs and files mixed
- [ ] `--depth N` (default 3, `0` = unlimited) counts levels after collapsing
- [ ] Bad `--sort`/`--depth` value: message plus short usage on stderr, exit 2
- [ ] Unit tests for each step in §6.1 order; goldens for `--files`, `--depth 1`, `--depth 0`, `--sort name`