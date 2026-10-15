# ctx

Shows how full the context window is as a bar, for example `ctx: ████░░░░░░ 42%`.

## Settings

| Key | Default | Meaning |
|---|---|---|
| `ctx.width` | `10` | Bar width in cells on a wide terminal (160 columns or more). |
| `ctx.width_medium` | `8` | Bar width below 160 columns. |
| `ctx.width_narrow` | `6` | Bar width below 120 columns. |
| `ctx.warn` | `50` | Usage percentage at or above which the fill turns amber. |
| `ctx.bad` | `90` | Usage percentage at or above which the fill turns red. |

## Depends on

- The session: the context window usage percentage, if the payload reports it.
- Its own config section and a painter for the label.
- A meter, which draws the bar and the percentage; the meter also reads the three width settings.

## Layout group

Identity.
