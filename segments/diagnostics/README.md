# diagnostics

Shows facts read from the session transcript: `boot` (`⌂ 64.5k`), `skills` (`⚙ 9.5k`), `skills_stripped` (`✗ skills 2/9`), `agent_skills` (`✗ agent skills 1/4`) and `mem` (`✗ mem`). It has no settings.

With `icons: nerd` the labels become Nerd Font icons: a rocket (U+F135) for `boot`, a gear (U+F013) for `skills`, and a warning triangle (U+F071) for `skills_stripped`, `agent_skills` and `mem`.

## Settings

None, so the package has no `defaults.yaml`.

## Depends on

- The transcript, as a function that is only called when one of these segments is drawn.
- A painter for the normal, amber and red text colours.
- A function that abbreviates a token count.

## Layout group

Usage, for all five segments.
