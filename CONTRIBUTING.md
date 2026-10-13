# Contributing

Thank you for helping with claude-statusline. This page covers what you need to change the code and get the change merged. For what the tool does and how a render works, start with the [README](README.md).

## Prerequisites

- Go, at the version in `go.mod`.
- [golangci-lint](https://golangci-lint.run/) v2, for the linters and formatters in `.golangci.yml` (the build uses v2.14.0).
- [shellcheck](https://www.shellcheck.net/), for the shell scripts under `hooks/`, `scripts/` and `tests/`.
- [GoReleaser](https://goreleaser.com/) and [syft](https://github.com/anchore/syft), only if you want to build a snapshot release locally.

`jq` is not needed: the shell scripts parse what they need with `awk` and `grep`, so the plugin works on a machine without it.

## Repository layout

| Path | What it is |
|---|---|
| `main.go` | Picks the command from the arguments (`--version`, `setup`, `subagent`, or the status line itself) and reads stdin. It never reports an error, because a blank status line is better than a broken one. |
| `internal/statusline` | Builds the line: loads config, saves the snapshot, merges rate limits, reads the transcript, runs hooks and renders. Also lists every config key the code reads. |
| `internal/segment` | The segments (`path`, `model`, `ctx`, `tokens`, `cache`, `rate5h`, `rate7d` and the transcript diagnostics) and the code that renders them in order and places custom segments among them. |
| `internal/layout` | Joins rendered segments into one row, or wraps them into groups when the terminal is too narrow. |
| `internal/config` | Layers your config file over `defaults.conf`, which is embedded in the binary. |
| `internal/payload` | Decodes the session JSON from Claude Code and strips control characters from its strings. |
| `internal/transcript` | Reads the few facts the line takes from the session transcript. |
| `internal/ratecache` | Shares rate-limit readings between sessions through a small cache file. |
| `internal/hook` | Runs custom segment scripts with a timeout and a per-session cache, and sanitises their output. |
| `internal/gitbranch` | Finds the branch by reading `.git` files, without running `git`. |
| `internal/bar` | Draws a usage bar with an optional pace marker. |
| `internal/theme` | Loads themes from YAML, compiles colour specs, and provides the colour-free variant used for `NO_COLOR`. The built-in themes are the files in `internal/theme/themes/`. |
| `internal/snapshot` | Saves the latest payload for other tools. |
| `internal/subagent` | Renders the rows of Claude Code's agent panel. |
| `internal/setup` | Adds `statusLine` and `subagentStatusLine` to `settings.json` once, backing the file up first. |
| `hooks/` | The plugin's `SessionStart` hook: `hooks.json` and `bootstrap.sh`, which installs the pinned release. |
| `scripts/` | Maintainer scripts. `pin-release.sh` records a release for the plugin to install. |
| `release/` | The pinned plugin version and the sha256 of each archive. |
| `tests/` | Shell tests for the bootstrap hook. |
| `testdata/` | Golden fixtures for the end-to-end tests. |
| `.claude-plugin/` | The plugin and marketplace manifests. |

## Verifying a change

CI runs the following on every pull request, and you can run the same commands locally.

```sh
test -z "$(gofmt -s -l .)"    # formatting
go vet ./...
go test -race ./...
golangci-lint run
shellcheck -s sh hooks/bootstrap.sh scripts/pin-release.sh tests/bootstrap_test.sh
sh tests/bootstrap_test.sh
goreleaser check
goreleaser release --snapshot --clean --skip=publish   # needs syft on your PATH
claude plugin validate .                               # needs Claude Code installed
```

The workflow files in `.github/workflows/` are the source of truth if this list and CI ever disagree. `golangci-lint run --fix` applies the formatters for you.

## Tests

Tests use [testify](https://github.com/stretchr/testify): `require` when the rest of the test cannot continue, `assert` otherwise. Test files live in external `_test` packages, so they exercise the public surface the way a caller would. Where a test needs to reach something unexported, the package has an `export_test.go` that exposes it to tests only, so production code gains no public API for the sake of testing.

### Golden fixtures

Most behaviour is pinned by golden fixtures. Each case is a directory holding the input and the exact output expected.

`testdata/golden/<case>/` is run through the status line by `golden_test.go`:

| File | Required | Purpose |
|---|---|---|
| `input.json` | yes | The session JSON Claude Code would send. |
| `expected` | yes | The exact output. |
| `config` | no | Written as the user's config file. |
| `env` | no | `KEY=value` lines added to the environment; lines starting with `#` are ignored. |
| `cache` | no | Seeds the rate-limit cache before the run. |
| `expected_cache` | no | The rate-limit cache contents expected after the run. |
| `tree` | no | Files to create in the temporary directory before the run, one `path=content` per line. A path ending in `/` makes a directory, a leading `+x ` makes the file executable (which custom segment scripts need) and `\n` in content is a newline. |

In `expected`, escape bytes are written as `\e`. `@TMP@` stands for the case's temporary directory in `input.json`, `config`, `tree` and `expected`. The run fixes the time (`STATUSLINE_NOW`) and a 200 column terminal unless the case's `env` says otherwise.

`testdata/golden-subagent/<case>/` is the same idea for the `subagent` command. It takes `input.json`, `expected`, and optionally `config` and `env`. Each expected row is the task id, a tab, then the row content with escapes written as `\e`.

Tests must never touch your real Claude Code state. The golden runners point `HOME` and `CLAUDE_CONFIG_DIR` at a temporary directory, and any test you add should do the same. When you run the binary by hand, do the same:

```sh
tmp=$(mktemp -d)
HOME="$tmp" CLAUDE_CONFIG_DIR="$tmp" go run . < testdata/golden/default_all/input.json
```

## Conventions

### Configuration keys

Every key the code reads must appear in `internal/config/defaults.conf` with a one-line comment above it, and every key in that file must be read by something. `TestKeysMatchDefaults` in `internal/statusline` enforces both directions, so a new setting cannot ship undocumented and a dead one cannot linger. Add the key to the table in the README too.

### Adding a built-in theme

Built-in themes are YAML files in `internal/theme/themes/`, embedded in the binary; the file name without `.yaml` is the name users put in `theme`. Write one in the format the README's Themes section describes, with comments, since the files double as examples. Start with `extends: classic` unless the theme needs every field. Parsing is strict (unknown keys are an error), so a typo fails the tests rather than shipping. Add a test in `internal/theme` for what is new, and a golden case under `testdata/golden/` that selects the theme in its `config`. Mention the theme in the README. Keep the loader's fallback to `classic` silent: a status line never prints an error.

### Adding a segment

1. Write a type that satisfies the `Segment` interface in `internal/segment`: a name, the layout group it belongs to when the line wraps, the config keys it reads, and a `Render` that returns an empty string when it has nothing to show. A segment that only reads the transcript is simpler as an entry in the `diagnostics` list in `diagnostics.go`.
2. Register it in the `registry` map in `internal/segment/segment.go`.
3. Add its keys to `defaults.conf`, and add its name to the default `segments` list if it should show by default.
4. Cover it with a unit test and at least one golden case, including the case where its data is missing.
5. Document it in the README, in the table of keys and the section that fits.

A segment should be hidden, not wrong, when its data is missing.

### Custom segment output

Scripts in the hook directory are user code, so their output is treated as untrusted. `internal/hook` strips control characters from it and keeps only colour sequences, so a script cannot move the cursor or redraw the terminal. If you change that path, keep the rule and extend its tests. The same applies to strings taken from the session payload, which `internal/payload` sanitises.

### Go style

- Methods on a struct use pointer receivers, never value receivers, so that a type has one consistent method set.
- Every receiver is named, with a short name taken from the type (`func (m *model) Keys()`, never `func (*model) Keys()`).
- Behaviour that operates on a struct's data is a method on that struct, so it is found by looking at the type rather than by searching the package for a free function. Constructors stay functions (`New`, `Load`, `Parse`, `Open`, `Discover`: anything that builds and returns a value for the caller to use), and so do stateless helpers with no natural owning type, such as `payload.Sanitize`, `segment.FormatTokens` and the colour-spec parser. Do not force a helper onto a type just to avoid a package-level function.

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

### Pinning a release for the plugin

The plugin installs the version recorded in `release/version`, not the latest release. After a release is published, a maintainer runs `scripts/pin-release.sh <version>` (for example `0.1.0`, without the leading `v`). It needs `gh`, verifies the build attestation and the checksum of all four archives, and only then writes `release/version` and `release/checksums.txt`. Commit both files in a pull request. Until that merges, the plugin keeps installing the previous pin, and while `release/version` is empty the hook does nothing.

The hashes are committed to the repository rather than read from the release on purpose. The hook compares each download with a hash that went through code review, so someone who could alter a release's files would still not get a tampered binary installed on users' machines. Reading the hash from the same place as the archive would only prove the two agree with each other.
