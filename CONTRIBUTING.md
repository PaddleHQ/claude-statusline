# Contributing

Thank you for helping with claude-statusline. This page covers what you need to change the code and get the change merged. For what the tool does, start with the [README](README.md); [How it works](docs/how_it_works.md) explains how a render runs, and [Customising](docs/customising.md) documents every setting.

## Prerequisites

- Go, at the version in `go.mod`.
- [golangci-lint](https://golangci-lint.run/) v2, for the linters and formatters in `.golangci.yml` (the build uses v2.14.0).
- [shellcheck](https://www.shellcheck.net/), for the shell scripts under `hooks/`.
- [GoReleaser](https://goreleaser.com/) and [syft](https://github.com/anchore/syft), only if you want to build a snapshot release locally.

`jq` is not needed: the shell scripts parse what they need with `awk` and `grep`, so the plugin works on a machine without it.

## Repository layout

| Path | What it is |
|---|---|
| `main.go` | Picks the command from the arguments (`--version`, `setup`, `subagent`, or the status line itself) and reads stdin. It never reports an error, because a blank status line is better than a broken one. |
| `internal/statusline` | Builds the line: loads config, saves the snapshot, merges rate limits, reads the transcript, runs hooks and renders. Also lists every config key the code reads. |
| `segments/` | One folder per plugin, each providing the segments it names and owning their `defaults.conf` and tests. `segments/all.go` is the single list of plugins. See [Building plugins](docs/building_plugins.md). |
| `internal/plugin` | What a plugin is: the `Segment` interface, the per-render `Session`, and the `Providers` (theme, config section, clock, and the lazily computed transcript and rate limits). |
| `internal/render` | Renders the configured segments in order, and places custom segments among them. |
| `internal/layout` | Joins rendered segments into one row, or wraps them into groups when the terminal is too narrow. |
| `internal/config` | Layers your config file over the defaults embedded in the binary: its own `defaults.conf` for the global keys, and each plugin's. |
| `internal/payload` | Decodes the session JSON from Claude Code and strips control characters from its strings. |
| `internal/transcript` | Reads the few facts the line takes from the session transcript. |
| `internal/ratecache` | Shares rate-limit readings between sessions through a small cache file. |
| `internal/hook` | Runs custom segment scripts with a timeout and a per-session cache, and sanitises their output. |
| `internal/gitbranch` | Finds the branch by reading `.git` files, without running `git`. |
| `internal/bar` | Draws a usage bar with an optional pace marker. |
| `internal/theme` | Loads themes from YAML, compiles colour specs, and provides the colour-free variant used for `NO_COLOR`. It reads the built-in themes from `themes/`. |
| `themes` | The built-in themes: one YAML file each, embedded in the binary by a tiny package that only exports them. |
| `internal/snapshot` | Saves the latest payload for other tools. |
| `internal/subagent` | Renders the rows of Claude Code's agent panel. |
| `internal/setup` | Adds `statusLine` and `subagentStatusLine` to `settings.json` once, backing the file up first. |
| `hooks/` | The plugin's `SessionStart` hook: `hooks.json` and `bootstrap.sh`, which installs the latest release, and `bootstrap_test.sh`, its shell tests. |
| `internal/rendertest` | Loads the txtar files the render tests and the image generator read. |
| `tools/screenshots` | The command that regenerates the documentation images: it runs each fixture through the status line and draws the output as SVG. It is one `main` package with its own tests. |
| `docs/` | The user documentation, and the generated images in `docs/images/`. |
| `testdata/` | The render test cases for the end-to-end tests, and the fixtures the documentation images are made from. |
| `.claude-plugin/` | The plugin and marketplace manifests. |

## Verifying a change

CI runs the following on every pull request, and you can run the same commands locally.

```sh
test -z "$(gofmt -s -l .)"    # formatting
go vet ./...
go test -race ./...
golangci-lint run
shellcheck -s sh hooks/bootstrap.sh hooks/bootstrap_test.sh
sh hooks/bootstrap_test.sh
goreleaser check
goreleaser release --snapshot --clean --skip=publish   # needs syft on your PATH
claude plugin validate .                               # needs Claude Code installed
```

The workflow files in `.github/workflows/` are the source of truth if this list and CI ever disagree. `golangci-lint run --fix` applies the formatters for you.

## Tests

Tests use [testify](https://github.com/stretchr/testify): `require` when the rest of the test cannot continue, `assert` otherwise. Test files live in external `_test` packages, so they exercise the public surface the way a caller would. Where a test needs to reach something unexported, the package has an `export_test.go` that exposes it to tests only, so production code gains no public API for the sake of testing.

### Render tests

Most behaviour is pinned by render tests. Each case is one [txtar](https://pkg.go.dev/golang.org/x/tools/txtar) file: a one-line description, then named sections holding the input and the exact output expected. This one is `testdata/render/branch_worktree_file.txtar`:

```txtar
A linked worktree, where .git is a file pointing at the real git directory, still shows its branch
-- input.json --
{"cwd":"@TMP@/wt"}
-- config --
segments = path
-- env --
NO_COLOR=1
-- tree --
wt/.git=gitdir: ../main/.git/worktrees/wt
main/.git/worktrees/wt/HEAD=ref: refs/heads/topic
-- output --
~/wt (topic)
```

`testdata/render/<case>.txtar` is run through the status line by `render_test.go` (`TestRender`), and the subtest is named after the file without `.txtar`. The text before the first section is the description of what the case shows; write one for every case. The sections are:

| Section | Required | Purpose |
|---|---|---|
| `input.json` | yes | The session JSON Claude Code would send. |
| `output` | yes | The exact output. |
| `config` | no | Written as the user's config file. |
| `env` | no | `KEY=value` lines added to the environment; lines starting with `#` are ignored. |
| `cache` | no | Seeds the rate-limit cache before the run. |
| `expected_cache` | no | The rate-limit cache contents expected after the run, compared ignoring surrounding whitespace. |
| `tree` | no | Files to create in the temporary directory before the run, one `path=content` per line. A path ending in `/` makes a directory, a leading `+x ` makes the file executable (which custom segment scripts need) and `\n` in content is a newline. |

In `output`, escape bytes are written as `\e`. `@TMP@` stands for the case's temporary directory in `input.json`, `config`, `tree` and `output`. The run fixes the time (`STATUSLINE_NOW`), a 200 column terminal and a 30 second hook timeout unless the case's `env` says otherwise.

A txtar section always ends in a newline, so the runner removes exactly one trailing newline from `output` (and from `expected_cache`, which is then compared without surrounding whitespace) before comparing. Everything else in `output` is compared exactly, including any further trailing newlines. An empty output is an empty section.

`testdata/subagent/<case>.txtar` is the same idea for the `subagent` command, run by `TestRenderSubagent`. It takes `input.json`, `output`, and optionally `config` and `env`. Each output row is the task id, a tab, then the row content with escapes written as `\e`.

When you change what the line looks like on purpose, rewrite the expected output instead of editing it by hand:

```sh
go test -run TestRender -update
```

`-update` rewrites only the `output` and `expected_cache` sections from the current code. The description and every other section stay as they are, and a case that already passes is not touched, so `git diff` shows exactly which outputs changed. Read that diff before you commit it. CI never passes `-update`.

Tests must never touch your real Claude Code state. The render runners point `HOME` and `CLAUDE_CONFIG_DIR` at a temporary directory, and any test you add should do the same. When you run the binary by hand, do the same:

```sh
tmp=$(mktemp -d)
echo '{"model":{"display_name":"Opus 4.7"}}' | HOME="$tmp" CLAUDE_CONFIG_DIR="$tmp" go run .
```

## Conventions

### Configuration keys

Every key the code reads must appear, with a one-line comment above it, in `internal/config/defaults.conf` (the global keys) or in the `defaults.conf` of the plugin that reads it, and every key in those files must be read by something. `TestKeysMatchDefaults` in `internal/statusline` enforces both directions, so a new setting cannot ship undocumented and a dead one cannot linger. Add the key to the table in [`docs/customising.md`](docs/customising.md) too.

### Adding a built-in theme

Built-in themes are YAML files in the top-level `themes/` directory, embedded in the binary by `themes/themes.go`; the file name without `.yaml` is the name users put in `theme`. Write one in the format the Themes section of [`docs/customising.md`](docs/customising.md) describes, with comments, since the files double as examples. Every theme is a complete, standalone file, so start by copying `themes/classic.yaml` or `themes/gradient.yaml`. Parsing is strict (unknown keys are an error), so a typo fails the tests rather than shipping. Add a test in `internal/theme` for what is new, and a render test case under `testdata/render/` that selects the theme in its `config`. Mention the theme in the Themes section of `docs/customising.md`. Keep the loader's fallback to `classic` silent: a status line never prints an error.

### Adding a segment

A segment is a plugin: a package under `segments/` that owns its code, its `defaults.conf` and its tests, plus one line in `segments/all.go`. [Building plugins](docs/building_plugins.md) is the step-by-step guide. A segment should be hidden, not wrong, when its data is missing.

### Custom segment output

Scripts in the hook directory are user code, so their output is treated as untrusted. `internal/hook` strips control characters from it and keeps only colour sequences, so a script cannot move the cursor or redraw the terminal. If you change that path, keep the rule and extend its tests. The same applies to strings taken from the session payload, which `internal/payload` sanitises.

### Documentation images

The pictures of the status line in the README and `docs/` are generated, not drawn. Each one comes from a txtar file, `testdata/screenshots/<name>.txtar`, which follows the same conventions as the render tests (an `input.json` section, and optionally `config`, `env` and `tree`, with the description before the first section; the run fixes the time and a 200 column terminal unless `env` says otherwise). It needs no `output`, because the image is the output. A `command` section containing `subagent` renders the agent panel rows instead of the status line. The generator runs the real status line against the fixture and writes `docs/images/<name>-dark.svg` and `docs/images/<name>-light.svg`, which the docs embed with a `<picture>` element so GitHub shows the one that matches the reader's theme.

After changing a fixture, or anything that changes what the line looks like, regenerate the images from the repository root and commit the result:

```sh
go run ./tools/screenshots
```

`TestScreenshotsUpToDate` regenerates every image in memory and fails when a committed one differs, is missing or has no fixture, so a stale picture cannot be merged.

### Go style

- Methods on a struct use pointer receivers, never value receivers, so that a type has one consistent method set.
- Every receiver is named, with a short name taken from the type (`func (m *model) Keys()`, never `func (*model) Keys()`).
- Behaviour that operates on a struct's data is a method on that struct, so it is found by looking at the type rather than by searching the package for a free function. Constructors stay functions (`New`, `Load`, `Parse`, `Open`, `Discover`: anything that builds and returns a value for the caller to use), and so do stateless helpers with no natural owning type, such as `payload.Sanitize`, `tokens.FormatTokens` and the colour-spec parser. Do not force a helper onto a type just to avoid a package-level function.

### Comments and language

Comments explain why, not what; code that reads clearly needs none. Code, identifiers and comments use US English. Documentation, commit messages and pull request text use British English.

## Pull requests and releases

Every pull request needs exactly one of these labels, checked by the `Check SemVer label` workflow:

| Label | Effect on merge |
|---|---|
| `release:major` | Tags the next major version and publishes it. |
| `release:minor` | Tags the next minor version and publishes it. |
| `release:patch` | Tags the next patch version and publishes it. |
| `norelease` | Merges without a release, for changes such as documentation or tests. |

On a pull request, a `release:*` label makes the release workflow do a dry run. On merge, the workflow tags the version and publishes archives for macOS and Linux on arm64 and amd64, with a checksum file, an SBOM per archive and a build provenance attestation.

### How the plugin picks up a release

Merging a `release:*` pull request publishes the release, and plugin users pick it up within a day: there is no second step. The hook resolves the latest release from the redirect at `https://github.com/PaddleHQ/claude-statusline/releases/latest`, so it does not use the API or its rate limit. It verifies the archive against the release's `checksums.txt` and, when `gh` is installed and logged in, runs `gh attestation verify`.

For tests only, `CLAUDE_STATUSLINE_LATEST_URL` replaces the latest-release URL (a `file://` URL whose content is the tag, for example `v1.2.3`), and `CLAUDE_STATUSLINE_DOWNLOAD_BASE` replaces the download base URL.
