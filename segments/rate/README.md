# rate

Shows usage of the 5 hour (`rate5h`) and 7 day (`rate7d`) windows as a bar with a pace marker, for example `5h: █▊┃░ 45%`.

## Settings

Each key exists for both segments, as `rate5h.<key>` and `rate7d.<key>`. The defaults below are for `rate5h`; `rate7d.warn` is `10` and `rate7d.bad` is `20`, with the label `7d`.

| Key | Default | Meaning |
|---|---|---|
| `rate5h.label` | `5h` | Text shown before the bar. |
| `rate5h.width` | `20` | Bar width in cells on a wide terminal (160 columns or more). |
| `rate5h.width_medium` | `14` | Bar width below 160 columns. |
| `rate5h.width_narrow` | `10` | Bar width below 120 columns. |
| `rate5h.warn` | `20` | Pace deviation, in percentage points, beyond which the fill turns amber. |
| `rate5h.bad` | `40` | Pace deviation, in percentage points, beyond which the fill turns red. |
| `rate5h.hard_cap` | `95` | Usage percentage at or above which the fill is always red. |
| `rate7d.label` | `7d` | Text shown before the bar. |
| `rate7d.width` | `20` | Bar width in cells on a wide terminal (160 columns or more). |
| `rate7d.width_medium` | `14` | Bar width below 160 columns. |
| `rate7d.width_narrow` | `10` | Bar width below 120 columns. |
| `rate7d.warn` | `10` | Pace deviation, in percentage points, beyond which the fill turns amber. |
| `rate7d.bad` | `20` | Pace deviation, in percentage points, beyond which the fill turns red. |
| `rate7d.hard_cap` | `95` | Usage percentage at or above which the fill is always red. |

## Depends on

- The rate-limit readings, as a function that is only called when a bar is drawn.
- The time of the render.
- Its own config section and a painter for the label.
- A meter, which draws the bar with its pace marker; the meter also reads the three width settings.

## Layout group

Rates.
