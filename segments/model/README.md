# model

Shows the model in use and its effort level, for example `Opus 4.7 [high]`. While a skill or subagent runs a different model it reads `Haiku (skill)`.

## Settings

| Key | Default | Meaning |
|---|---|---|
| `model.effort` | `true` | Show the effort level after the model name. |

## Depends on

- The session: the model's display name and id, the effort level and the transcript path.
- The transcript, as a function that is only called when the payload names a transcript, to find the model that is really running.
- Its own config section, and a painter for the text colour.

## Layout group

Identity.
