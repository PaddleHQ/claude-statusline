# claude-statusline

A small, configurable status line for Claude Code. It shows the model and your
subscription usage as bars with a pace marker:

```
Opus 4.7 [high] · 5h: █████████░┃░░░░░░░░░ 45% · 7d: ████████░░┃░░░░░░░░░ 40%
```

## Install

Requires `jq`. Copy `statusline.sh` somewhere stable, for example
`~/.claude/claude-statusline/statusline.sh`, then add this to
`~/.claude/settings.json`:

```json
"statusLine": {
  "type": "command",
  "command": "sh ~/.claude/claude-statusline/statusline.sh",
  "refreshInterval": 60
}
```

If `jq` is missing the status line prints nothing. If you set
`CLAUDE_CONFIG_DIR`, the script reads and writes under that directory instead
of `~/.claude`.

## Configuration

Optional. Create `~/.claude/claude-statusline/config` with `key = value` lines.
Blank lines and lines starting with `#` are ignored, whitespace around keys and
values is trimmed, and the last occurrence of a key wins. The file is parsed,
never executed, so values are always treated as plain text. Invalid numbers fall
back to the default.

Each segment's keys are written `<segment>.<key>`. Keys must be lower case
with at most one dot; anything else is ignored.

| Key | Default | Meaning |
|---|---|---|
| `segments` | `model rate5h rate7d` | Segments to show, in order. Omit one to drop it; unknown names are skipped. |
| `separator` | `·` | Glyph between segments. |
| `theme` | `default` | Colour theme. Unknown names fall back to `default`. |
| `model.effort` | `true` | `false` hides the `[effort]` suffix. |
| `rate5h.label`, `rate7d.label` | `5h`, `7d` | Text before the bar. |
| `rate5h.width`, `rate7d.width` | `20` | Bar width in cells on a wide terminal. |
| `rate5h.warn`, `rate5h.bad` | `20`, `40` | Pace deviation (percentage points) for amber and red. |
| `rate7d.warn`, `rate7d.bad` | `10`, `20` | The same for the 7 day window. |
| `rate5h.hard_cap`, `rate7d.hard_cap` | `95` | Usage at or above this is always red. |

Bars narrow on small terminals: half width below 120 columns and 70% below 160
(`$COLUMNS`), never under 4 cells.

### Colour

Colours use the terminal's own palette so they read on light and dark
backgrounds. Set `NO_COLOR` to any non-empty value to turn colour and styling
off.

## How the bars work

The vertical marker `┃` sits at the point in the window you would reach if you
spent your quota evenly. The fill is green while your usage is close to the
marker, amber when it drifts past `warn` points either way, and red past `bad`
points or at the hard cap. A bar disappears once its window has reset.

Rate limits belong to the account, not the session, and an idle session's
payload goes stale. Readings are therefore shared through
`~/.claude/claude-statusline/cache/ratelimits`. A session only replaces the
cached reading with a newer window, higher usage, or a drop of more than 10
points (a mid-window quota reset), and the line is always drawn from the merged
value.

## Development

```sh
sh tests/run.sh
shellcheck -s sh statusline.sh tests/run.sh
```

Tests are golden files under `tests/cases/<name>/`: `input.json`, `expected`,
and optionally `config`, `cache`, `env` and `expected_cache`. In `expected`,
the escape byte is written as `\e`; in `cache` files `|` stands for the
separator byte. Set `STATUSLINE_NOW` to a Unix timestamp to fix the clock
(the test runner does).

## Licence

MIT. See `LICENSE`.
