# cache

Shows how much of the latest request's input came from the prompt cache, for example `⧉ 80%`. It turns red when the figure is low.

## Settings

| Key | Default | Meaning |
|---|---|---|
| `cache.warn` | `5` | Prompt cache hit percentage below which the cache figure turns red. |

## Depends on

- The session: the token usage of the latest request.
- Its own config section and a painter for the normal and red text colours.

## Layout group

Usage.
