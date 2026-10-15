---
id: kiset3
status: open
type: feature
effort: normal
created: 2026-10-15T18:56:58Z
---
# Add --watch to refresh the display

`--watch[=DUR]` reruns measure + render on an interval and redraws the screen.

- Default 5s; `--watch=DUR` takes Go durations (`500ms`, `5s`, `1m`), must be > 0, else exit 2.
  `--watch 5s` does not work: `5s` becomes the revision.
- Polls on a timer; no fsnotify, no new dependencies.
- All modes; revision/range is re-resolved each tick.
- Each frame: one stdout write of clear screen + cursor home + output. Tall output scrolls.
- Header gains ` · every 5s · 15:04:05` (local time) after the filter suffix; never trimmed.
- Non-TTY stdout: usage error, exit 2, `--watch needs a terminal`.
- First-tick errors exit as today; later errors show `blast-radius: <err>` as the frame and retry.
- Warnings: no special code; they go to stderr and the next clear wipes them.
- Ctrl-C: default signal handling, last frame stays.
- Spec: relax the TUI non-goal, flags table, help, §7.2 header, §8, §10.
- Tests: non-TTY exit 2, bad durations, flag parsing unit test; loop checked by hand.