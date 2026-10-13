# claude-statusline

A small, configurable status line for Claude Code. It shows the model and your
subscription usage as bars with a pace marker:

```
Opus 4.7 [high] · 5h: █████████░┃░░░░░░░░░ 45% · 7d: ████████░░┃░░░░░░░░░ 40%
```

## Install

Requires `jq`. Copy the whole directory (`statusline.sh`, `defaults.conf`,
`lib/`, `segments/` and `themes/`) to `~/.claude/claude-statusline/`, then add
this to `~/.claude/settings.json`:

```json
"statusLine": {
  "type": "command",
  "command": "sh ~/.claude/claude-statusline/statusline.sh",
  "refreshInterval": 60
}
```

Your own settings live in `~/.claude/claude-statusline/config`, the same
directory. This project never ships a file called `config`, so copying a newer
version over the directory leaves your settings alone. The rate-limit cache is
written to `cache/` there too.

If `jq` or `defaults.conf` is missing the status line prints nothing. If you set
`CLAUDE_CONFIG_DIR`, everything is read from and written under that directory
instead of `~/.claude`.

## Configuration

Optional. `defaults.conf` lists every setting with a comment and its default
value. Copy the lines you want to change into
`~/.claude/claude-statusline/config` as `key = value`. Blank lines and lines
starting with `#` are ignored, whitespace around keys and values is trimmed, and
the last occurrence of a key wins. The file is parsed, never executed, so values
are always plain text. An invalid value (a non-number where a number is
expected, for instance) falls back to the default.

Keys are `<segment>.<key>`, in lower case with at most one dot; anything else is
ignored. The main ones:

| Key | Meaning |
|---|---|
| `segments` | Segments to show, in order. Omit one to drop it; unknown names are skipped. |
| `separator` | Glyph between segments. |
| `theme` | Colour theme. Unknown names fall back to `default`. |

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

## Adding a segment

1. Create `segments/<name>.sh` (lower case letters, digits and underscores)
   defining `segment_<name>`. It prints its text, or nothing to hide itself.
   Payload fields are available as globals such as `MODEL_NAME`; use `paint` and
   the theme colours rather than raw escape codes.
2. Read settings with `config_text`, `config_number` or `config_bool` using
   `<name>.<key>`, and add each key to `defaults.conf` with a one-line comment.
   There are no default arguments in code.
3. Add the name to `segments` in `defaults.conf` if it should be on by default,
   and golden tests under `tests/cases/`.

`sh tests/check_defaults.sh` fails if code and `defaults.conf` disagree.

## Development

```sh
sh tests/run.sh
sh tests/check_defaults.sh
shellcheck -s sh -x statusline.sh segments/*.sh themes/*.sh tests/*.sh
```

Tests are golden files under `tests/cases/<name>/`: `input.json`, `expected`,
and optionally `config`, `cache`, `env`, `expected_cache` and `remove` (files
deleted from a copy of the install before the run). In `expected`,
the escape byte is written as `\e`; in `cache` files `|` stands for the
separator byte. Set `STATUSLINE_NOW` to a Unix timestamp to fix the clock
(the test runner does).

## Licence

MIT. See `LICENSE`.
