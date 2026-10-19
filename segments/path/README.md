# path

Shows the working directory, shortened, with the git branch when there is one, for example `~/code/app (main)`. A git worktree is shown as the name of its main repository.

## Settings

| Key | Default | Meaning |
|---|---|---|
| `path.strip` | `[]` | Directory prefixes to strip, as a list; a leading `~` means your home directory. |
| `path.branch` | `true` | Show the git branch after the path. |
| `path.worktrees` | `[]` | Directories that hold git worktrees, as a list; relative entries are relative to the main repository and a leading `~` means your home directory. Empty means off. |
| `path.emoji` | `false` | Show the theme's emoji before the branch. |

With `worktrees: [.worktrees]`, a session in `~/code/app/.worktrees/fix` shows `app (fix)`, and one in a subdirectory of it shows `app/pkg/sub (fix)`. The branch of a worktree has its own colour, and `strip` is ignored for it. The worktree's own root has to sit under a listed directory, so the main repository itself is shown as usual.

## Depends on

- The session: the working directory.
- Its own config section, the home directory and a painter for the path colour.
- A git reader, which gives the printable branch name of a directory and, inside a linked worktree, its root and its main repository.
- The painter's branch and worktree colours and the theme's two emoji.

## Layout group

Identity.
