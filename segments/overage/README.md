# overage

Shows the extra-usage spend this calendar month, for example `» ~$5.00`. It is off by default, through the default `segments_excluded: [cost, overage]`; set `segments_excluded: []` to show it. It is never hidden: with nothing spent it is dimmed, as `» ~$0.00`.

Claude Code reports no flag for a request that was paid as extra usage, so the figure is inferred. While your 5 hour or 7 day window is at 100% and requests still go through, the cost each session adds (Claude Code's own estimate at API prices) is added to the month's total, which is shared between sessions and resets at 00:00 UTC on the 1st. It is an estimate, not a bill. It is amber while a window is at 100% and the figure is above zero.

Logins that report a spend limit show it instead, as `» $12.50/$50`, or `» 40%` when only the percentage is known. It is amber once anything is spent and dim at zero.

With `icons: nerd` the » becomes the Nerd Font warning icon (U+F071).

## Settings

None, so the package has no `defaults.yaml`.

## Depends on

- The session: the reported spend limit, as a percentage or in dollars, either of which may be missing.
- The shared estimate of the month's spend, which the status line updates on every render.
- The account's usage windows, to tell whether one is at 100%.
- A painter for the text, amber and dim colours, and the icon set for the label.

## Layout group

Usage.
