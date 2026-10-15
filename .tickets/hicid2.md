---
id: hicid2
status: closed
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

## Note 2026-10-15T18:14:08Z

Done. tree.Shape(root, Options{Files, ByName, Depth}) runs the §6.1 steps in order, mutating in place; each step is exported (HideFiles, Collapse, Sort, LimitDepth) and unit-tested. Collapsed nodes keep the OUTER dir's totals (hidden files still count); only dir->dir merges (spec §6.1 step 2 clarified). Render is unchanged: it prints whatever children remain, so the bar ticket's maxChurn = max over the shaped tree's nodes.
- main.go: --depth (fs.Int), --files, --sort (fs.String) validated after Parse via a local usageError(format, ...) helper: "blast-radius: <msg>" + fs.Usage(), exit 2. Flags are validated before any git call. Usage text is still the flag package default (lazim7 owns help).
- e2e: shapeRepo(t) fixture (nested dirs, chains, equal-churn ties) + goldens shape-default/-files/-depth-1/-depth-0 (with --files, 4 levels)/-sort-name; uncommitted.golden regenerated (now dirs only, churn-sorted). TestBadFlagValues asserts messages without a golden so usage changes don't churn goldens.