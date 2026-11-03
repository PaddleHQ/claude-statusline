# cost

Shows what the session would cost at API prices, in US dollars, for example `$1.23`. It turns amber and then red as the figure grows, and is hidden when Claude Code does not report a cost. It is off by default, through the default `segments_excluded: [cost]`; set `segments_excluded: []` to show it.

Claude Code reports this figure: its own estimate of the session's tokens priced at the public API rates. What it means depends on how you pay:

- **API key (pay as you go):** it is roughly what the session has cost you. Your bill is the authority, so treat the figure as an estimate.
- **Pro or Max subscription:** you pay a flat fee, so the figure is notional. It is what the same work would have cost on the API, not money you are charged, and it does not count against anything. It is still useful for comparing sessions and for spotting an expensive one. What limits a subscription is your 5 hour and 7 day usage, which `rate5h` and `rate7d` show.

## Settings

| Key | Default | Meaning |
|---|---|---|
| `cost.warn` | `5` | Session cost in whole US dollars at or above which the figure turns amber. |
| `cost.bad` | `20` | Session cost in whole US dollars at or above which the figure turns red. |

## Depends on

- The session: its total cost.
- Its own config section and a painter for the normal, amber and red text colours.

## Layout group

Usage.
