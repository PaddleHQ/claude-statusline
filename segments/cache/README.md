# cache

Shows how much of the latest request's input came from the prompt cache, for example `⧉ 80%`. It turns red when the figure is low. Once the cache has expired since the last response, as in a session resumed after a break, it shows `⧉ cold` in red instead.

With `icons: nerd` the `⧉` becomes the Nerd Font bolt icon (U+F0E7).

## Settings

| Key | Default | Meaning |
|---|---|---|
| `cache.warn` | `5` | Prompt cache hit percentage below which the cache figure turns red. |

## Depends on

- The session: the token usage of the latest request, and the transcript path.
- The transcript: when the latest response was written and how long its cache lives (an hour when it wrote only to the one-hour cache, five minutes otherwise).
- The current time, its own config section and a painter for the normal and red text colours.

## Layout group

Usage.
