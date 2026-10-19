# path

Shows the working directory, shortened, with the git branch when there is one, for example `~/code/app (main)`.

## Settings

| Key | Default | Meaning |
|---|---|---|
| `path.strip` | `[]` | Directory prefixes to strip, as a list; a leading `~` means your home directory. |
| `path.branch` | `true` | Show the git branch after the path. |

## Depends on

- The session: the working directory.
- Its own config section, the home directory and a painter for the path colour.
- A git reader, which gives the printable branch name of a directory.

## Layout group

Identity.
