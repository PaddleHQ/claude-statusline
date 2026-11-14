# ctx

Shows how full the context window is as a bar, for example `ctx: ████░░░░░░ 42%`.

A marker on the bar shows where to run `/compact` yourself before Claude Code compacts for you. Once the context tokens reach that point the fill turns red and `⚠ /compact` follows the percentage. Below it the amber and red thresholds apply as usual.

The marker sits at the lower of the two limits `ctx.compact_at` and `ctx.compact_pct`; set one to `0` to ignore it, or both to `0` for the plain bar. Context tokens are the latest request's input, cache creation and cache read tokens. The window size comes from the payload, or is worked out from the usage percentage when the payload omits it. With no window size the percentage limit cannot apply and no marker is drawn, though the warning still shows once the tokens reach `ctx.compact_at`. A limit at or beyond the end of the window draws no marker.

With `icons: nerd` the label `ctx:` becomes the Nerd Font memory chip icon (U+F035B).

## Settings

| Key | Default | Meaning |
|---|---|---|
| `ctx.width` | `10` | Bar width in cells on a wide terminal (160 columns or more). |
| `ctx.width_medium` | `8` | Bar width below 160 columns. |
| `ctx.width_narrow` | `6` | Bar width below 120 columns. |
| `ctx.warn` | `50` | Usage percentage at or above which the fill turns amber. |
| `ctx.bad` | `90` | Usage percentage at or above which the fill turns red. |
| `ctx.compact_at` | `200000` | Context tokens at which the marker and the `/compact` warning apply; `0` turns this limit off. |
| `ctx.compact_pct` | `80` | Percentage of the context window at which the marker and the warning apply; `0` turns this limit off. |

## Depends on

- The session: the context window usage percentage, if the payload reports it, plus the window size and the latest request's token counts.
- Its own config section and a painter for the label and the warning.
- A meter, which draws the bar and the percentage; the meter also reads the three width settings and draws the compact marker.

## Layout group

Identity.
