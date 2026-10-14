# tokens

Shows the session's token totals, for example `↓ 1.5k · ↑ 23.4k`. It has no settings.

## Settings

None, so the package has no `defaults.conf`.

## Depends on

- The session: the total input and output tokens, each of which may be missing.
- A painter for the text colour.
- The painted separator to put between the two totals.

The package also exports `FormatTokens`, a pure helper that abbreviates a count. The composition root hands it to the diagnostics plugin.

## Layout group

Usage.
