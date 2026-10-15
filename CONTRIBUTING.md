# Contributing

Thank you for helping with claude-statusline. This page covers what you need to change the code and get the change merged. For what the tool does, start with the [README](README.md).

## Prerequisites

- Go, at the version in `go.mod`.
- [golangci-lint](https://golangci-lint.run/) v2, for the linters and formatters in `.golangci.yml` (the build uses v2.14.0).
- [shellcheck](https://www.shellcheck.net/), for the shell scripts under `hooks/`.
- [GoReleaser](https://goreleaser.com/) and [syft](https://github.com/anchore/syft), only if you want to build a snapshot release locally.

`jq` is not needed: the shell scripts parse what they need with `awk` and `grep`, so the plugin works on a machine without it.

## Repository layout

| Path | What it is |
|---|---|
| `main.go` | A few lines of wiring: it holds the build stamps and hands them, the arguments and the standard streams to `internal/cli`. It has no test, so that its wiring does not count against coverage. |
| `internal/cli` | Picks the command from the arguments (`--version`, `setup`, `subagent`, or the status line itself) and reads stdin. It never reports an error, because a blank status line is better than a broken one. |
| `internal/statusline` | Builds the line: loads config, saves the snapshot, merges rate limits, reads the transcript and renders. Also lists every config key the code reads. |
| `segments/` | One folder per plugin, each providing the segments it names and owning their `defaults.conf`, README, tests and generated mocks. A plugin declares the interfaces it needs and imports nothing from `internal/`. `segments/all.go` is the composition root: the single list of plugins, and the environment that hands them the internal types, which satisfy the plugins' interfaces directly. |
| `internal/render` | Renders the configured segments in order. |
| `internal/layout` | Joins rendered segments into one row, or wraps them into groups when the terminal is too narrow. |
| `internal/config` | Layers your config file over the defaults embedded in the binary: its own `defaults.conf` for the global keys, and each plugin's. |
| `internal/payload` | Decodes the session JSON from Claude Code and strips control characters from its strings. |
| `internal/transcript` | Reads the few facts the line takes from the session transcript. |
| `internal/ratecache` | Shares rate-limit readings between sessions through a small cache file. |
| `internal/gitbranch` | Finds the branch by reading `.git` files, without running `git`. |
| `internal/bar` | Draws a usage bar with an optional pace marker. |
| `internal/theme` | Loads themes from YAML, compiles colour specs, and provides the colour-free variant used for `NO_COLOR`. It reads the built-in themes from `themes/`. |
| `themes` | The built-in themes: one YAML file each, embedded in the binary by a tiny package that only exports them. |
| `internal/snapshot` | Saves the latest payload for other tools. |
| `internal/subagent` | Renders the rows of Claude Code's agent panel. |
| `internal/setup` | Adds `statusLine` and `subagentStatusLine` to `settings.json` once, backing the file up first. |
| `hooks/` | The plugin's `SessionStart` hook: `hooks.json` and `bootstrap.sh`, which installs the latest release, and `bootstrap_test.sh`, its shell tests. |
| `features/` | The Gherkin feature files: readable examples of the whole line. `features_test.go` runs them. |
| `internal/featuretest` | The step definitions behind the feature files. |
| `.claude-plugin/` | The plugin and marketplace manifests. |

## Verifying a change

CI runs the following on every pull request, and you can run the same commands locally.

```sh
go mod verify
go mod tidy -diff             # go.mod and go.sum are tidy
! grep -q '^toolchain ' go.mod   # no toolchain directive
test -z "$(gofmt -s -l .)"    # formatting
go generate ./... && git diff --exit-code   # generated mocks are up to date
go vet ./...
go test -race ./...
# coverage at or above requiredCoverage in build-config.yaml: see Coverage below
golangci-lint run
shellcheck -s sh hooks/bootstrap.sh hooks/bootstrap_test.sh
sh hooks/bootstrap_test.sh
actionlint                    # the workflow files
editorconfig-checker          # whitespace and indentation against .editorconfig
goreleaser check
goreleaser release --snapshot --clean --skip=publish   # needs syft on your PATH
claude plugin validate .                               # needs Claude Code installed
```

The workflow files in `.github/workflows/` are the source of truth if this list and CI ever disagree. `golangci-lint run --fix` applies the formatters for you.

Dependabot opens weekly pull requests for Go modules (labelled `release:patch`) and GitHub Actions (labelled `norelease`), so they pass the SemVer label check. A pull request carrying a blocking label, such as `do not merge`, fails the "Check for blocking labels" check until the label is removed.

## Tests

Tests use [testify](https://github.com/stretchr/testify): `require` when the rest of the test cannot continue, `assert` otherwise. Test files live in external `_test` packages, so they exercise the public surface the way a caller would. Where a test needs to reach something unexported, the package has an `export_test.go` that exposes it to tests only, so production code gains no public API for the sake of testing.

### Which layer does my test belong in?

There are three layers. Put a test in the lowest one that can show the behaviour, and prefer table-driven tests (`t.Run` over a slice of cases) in the two unit layers.

**1. Plugin unit tests**, in `segments/<plugin>/*_test.go`, with [mockery](#mocks) mocks. They cover every edge of what one plugin draws: how it formats, where its thresholds and colours change, what it does when data is missing, and how each config variant changes it. A plugin test never builds a payload or a theme; the mocks stand in for them. For example:

- `tokens` abbreviating 999 as `999`, 1000 as `1.0k` and 1500000 as `1.5M`.
- `cache` turning red below `cache.warn`, and hiding when the counts add up to zero.
- `model` showing `Haiku (skill)` when the transcript says a different family ran.

**2. Provider unit tests**, in `internal/*` (and `segments/render_test.go` for the composition root). They cover the shared machinery that the plugins sit on: config layering and validation, theme loading, fallback and colour specs, the rate-limit cache's freshness rules, the transcript's bounded reads, layout and wrapping, setup, the snapshot and the bars. They use real files in a temporary directory, never your `HOME`. For example:

- a theme file with an unknown colour name falls back to `classic` (`internal/theme`).
- a reading ten points below the cached one in the same window is stale, and eleven is a reset (`internal/ratecache`).

**3. Gherkin features**, in `features/`. They show the overall shape of the line in examples a reader can follow: the default line, segments dropped or reordered, wrapping at a few widths, each theme applied once, `NO_COLOR`, a couple of subagent rows, and one example for each plugin so that every feature file still documents its segment. For example:

- the default line with a full payload.
- the line wrapping at 80 and 40 columns, with a continuation arrow.

Ask of a new test: is it about one plugin's output (layer 1), about how the program's parts combine or what a shared part does with its input (layer 2), or would a reader learn from seeing the whole line (layer 3)? A feature scenario that pins a colour threshold, a boundary value or an error fallback belongs in a unit test instead. The relevant unit tests also have to exist before a scenario is deleted, so that no behaviour is pinned only by a scenario that was removed.

### Coverage

CI's Coverage job runs the tests with `-coverpkg ./...`, so a package is credited for statements that other packages' tests run, and fails when total statement coverage is below `requiredCoverage` in `build-config.yaml`. It leaves out generated mocks (`/mocks/`), `tools/` and the feature-test harness (`internal/featuretest`), and writes a per-package table to the job summary. `main` is kept free of tests, because a test file there would pull its untested wiring into the total. When coverage is a point or more above the requirement, the job prints a notice with the value to raise it to; raising it is a ratchet, so do it in the pull request that earned it. To reproduce the job locally:

```sh
mkdir -p covdata
go test -race -cover -covermode=atomic -coverpkg ./... ./... -args -test.gocoverdir="$PWD/covdata"
go tool covdata textfmt -i=covdata -o cover.out
grep -vE '/mocks/|/tools/|/internal/featuretest/' cover.out > cover.filtered.out
go tool cover -func cover.filtered.out | tail -n 1
```

### Fuzzing

The status line runs on every render and reads input it does not control: the session JSON, the transcript, the rate-limit cache, and your config and theme files. Each of those parsers has a native Go fuzz target (`FuzzParse`, `FuzzRead` and so on, in the `fuzz_test.go` of its package, or beside the unit tests for `layout.Width` and `tokens.FormatTokens`). A target checks properties rather than only that nothing panics: payload strings hold no control characters, a colour spec compiles to colour sequences only, a transcript read stays within its byte bounds, and so on.

`go test ./...` runs every target on its seed inputs, so they are ordinary tests in CI. To let the fuzzer search for new inputs, run one target at a time:

```sh
go test -run='^$' -fuzz='^FuzzParse$' -fuzztime=60s ./internal/payload
```

The "Weekly fuzz" workflow does the same for every target on a schedule, and can be started by hand from the Actions tab. When a run fails, download the `crashers-*` artefact: it holds the new input as `testdata/fuzz/<Target>/<id>`. Copy it into the same path in your checkout, run `go test -run='FuzzParse/<id>' ./internal/payload` to reproduce it, and fix the code (or the property, if the property was wrong). Commit the input file with the fix, so it stays as a regression test.

### Mocks

Plugin tests use mocks generated by [mockery](https://vektra.github.io/mockery/) v2, which is a `tool` in `go.mod`, so nothing needs installing. Each interface a plugin declares has a `//go:generate go run github.com/vektra/mockery/v2 --name=<Interface>` line above it, and `.mockery.yaml` holds the options. The mocks are written to a `mocks/` folder in the plugin's package and are committed. After adding or changing a plugin interface, regenerate them from the repository root:

```sh
go generate ./...
```

CI runs the same command and fails if it changes anything, so a stale mock cannot be merged. Do not edit the files in `mocks/`. Tests use the typed expecter API, for example `m.EXPECT().DisplayName().Return("Opus 4.7")`.

### Feature tests

The feature tests are [Gherkin](https://cucumber.io/docs/gherkin/) files under `features/`, run by [godog](https://github.com/cucumber/godog). Each file covers one topic (`wrapping.feature`, `rate_bars.feature`, and so on), and each scenario shows one example, named after what it shows. They keep to the composition and the readable examples; see the layers above for what belongs in a unit test instead. This one is in `features/path_and_branch.feature`:

```gherkin
Scenario: The branch is found by walking up from a nested working directory to the repository's .git
  Given the session:
    """json
    {
      "cwd": "@TMP@/repo/pkg/sub"
    }
    """
  And the config:
    """
    segments = path
    """
  And the environment variable "NO_COLOR" is "1"
  And the file "repo/.git/HEAD":
    """
    ref: refs/heads/main
    """
  And the directory "repo/pkg/sub"
  When the status line renders
  Then the text is:
    """
    ~/repo/pkg/sub (main)
    """
```

`go test ./...` runs every feature file as `TestFeatures` in `features/features_test.go`: each scenario is a Go subtest named after it, and the run is strict, so a step that matches nothing fails the test instead of being skipped. All the steps live in `internal/featuretest`, and the vocabulary is small and fixed, so a new scenario is written entirely in the `.feature` file and needs no Go code.

#### Steps

A step that takes text ends in a colon and is followed by a `"""` block. The text of every `Given` is written before the `When`, which runs the command once; the `Then` steps check what it did.

| Step | What it does |
|---|---|
| `Given the session:` | The session JSON that Claude Code sends on standard input. It may be empty or invalid. |
| `Given the config:` | Writes the user's config file. |
| `Given the environment variable "NAME" is "value"` | Sets one environment variable for the run. |
| `Given the file "path":` | Creates a file at `path` under the temporary home directory, with the block as its content and a final newline. Directories are created as needed. A transcript is any file the session names. |
| `Given the directory "path"` | Creates an empty directory. |
| `Given the rate-limit cache:` | Seeds the shared rate-limit cache file before the run. |
| `When the status line renders` | Runs the status line. |
| `When the status line renders at <n> columns` | The same, with the terminal `<n>` columns wide. |
| `When the subagent rows render` | Runs the `subagent` command. |
| `Then the output is:` | The output with its colours and styles, written as rows of text with a style line under each; see [Expected output](#expected-output). |
| `Then the text is:` | The output with every colour and style removed. Use it when the scenario has no colour at all. |
| `Then the raw output is:` | The exact bytes, with each escape byte written as `\e`. Use it only when the bytes are the point. |
| `Then the rate-limit cache is:` | The cache file's content after the run, compared ignoring surrounding whitespace. |

In every block, `@TMP@` stands for the scenario's temporary directory, which is also `HOME` and `CLAUDE_CONFIG_DIR`, so tests never touch your real Claude Code state. For `the subagent rows render`, every row of the output is the task id, `\t` for the tab, then the row content, and `\t` counts as two columns. In a `Given the session:` block, write the JSON indented, with the keys in the order the payload has them. An empty output is an empty block. A line break is only the separator between lines, so the block has no trailing newline unless you add an empty last line.

#### Expected output

`Then the output is:` shows each row of output as a line of text, with a line under it that gives one style character for each column of the text above. This one is the five hour bar at 75% used, drawn in yellow with the bold pace marker and the dim unused part:

```text
5h: ██████████┃████░░░░░ 75%
    yyyyyyyyyyByyyddddd
```

The style character says which styles were in effect when that column was printed. A space means no styling. These letters are built in:

| Character | Style | Character | Style |
|---|---|---|---|
| `d` | dim | `B` | bold |
| `r` | red | `g` | green |
| `y` | yellow | `b` | blue |
| `m` | magenta | `c` | cyan |
| `w` | white | `k` | bright black |

Everything else needs a legend: true colours, 256 colour palette entries, backgrounds, the other bright colours, black and anything combined, such as bold yellow. The legend follows the rows after one empty line, and has a line for each style in the form `<character> = <style>`. The character is a digit or a letter that is not built in, and a built-in character cannot be redefined. The style is one or more of `bold`, `dim`, a colour (`red`, `bright-red` and so on, `#rrggbb`, or `256:n` for a palette entry) and the same prefixed with `bg:` for a background, separated by spaces:

```text
ctx: ▰▰▰▰▰▰▱▱▱▱ 62%
     1234567777 yyy

1 = #94e2ad
2 = #aae2ad
3 = #c0e2ad
4 = #d7e2ae
5 = #ede2ae
6 = #f9d9af
7 = #4a4a4a
```

Some more rules:

- Every row has a style line, even a row with no styling, which has an empty one. A blank line in the place where a text row would start is what begins the legend, so a text row itself cannot be empty in an output that has styles. If nothing in the output has a style, use `Then the text is:` instead.
- A wide character, one that takes two columns such as an East Asian character or an emoji, is written with its style character and then `_` under it, or with two spaces if it has no style. Characters that take no column, such as a zero-width joiner, share the column of the character before them. The status line itself counts every character as one column; the widths only keep the style line aligned in your editor.
- Trailing spaces are not significant, so a short style line means the columns after it have no styling. The comparison also ignores unstyled spaces at the end of a row. A styled space at the end of a row is kept: write it in the text line, or leave the text line short and the style line long.
- The comparison is of the style in effect for each column, not of the bytes, so a redundant reset or a repeated code does not matter. The output may only use colour and style sequences the harness understands (reset, bold, dim, the 16 colours, 256 colour and true colour, for the foreground and the background). Any other escape sequence, or any style code it does not know, fails the scenario with a message that names it, and a scenario like that belongs on `Then the raw output is:`.
- The scenario's temporary directory is written `@TMP@`, as in a `Given` block, in the output as well. It is five columns wide.

There is no update flag. When the output differs, the failure prints the expected rows, the actual rows in this same format ready to paste, and a line by line diff, so you can read what changed and copy the new block once you have checked that it is right.

Pick the step by what the scenario is about. Most scenarios use `Then the output is:`. A scenario with no colour, such as one that sets `NO_COLOR` or an empty output, uses `Then the text is:`. `Then the raw output is:` is for the few scenarios where the exact bytes are the point, for example one that checks how escape sequences or control characters in input are passed on or removed.

Every scenario starts with the fixed time (`STATUSLINE_NOW`) and a 200 column terminal, set by the `Background` of the feature files that render the status line. A scenario's own steps override them. Whatever the command prints, the run also fails if the output holds a control character other than newline and escape, or if a config value was executed.

There are no domain-specific steps. Where a scenario would repeat itself for several values, that is usually a sign the cases belong in a table-driven unit test; if a feature does need the same steps with different values, a `Scenario Outline` substitutes the `<placeholders>` of its `Examples` table into them. Add a domain step only if the scenario would otherwise be unreadable, and document it in the table above.

#### Adding a scenario

1. Check that it is an example or a composition (layer 3 above). The edges of one plugin or of a shared part go in a unit test, with the plugin's mocks or the package's own helpers.
2. Pick the feature file for the behaviour, or add a new `features/<topic>.feature` with a `Feature:` title, a line or two of description and a `@<topic>` tag. Copy the closest scenario, and give it a name that says what it shows. Keep to one behaviour per scenario, and add a new scenario rather than another `Then` to an old one.
3. Run it, and write the expected output by hand. There is no update flag: when the output differs, the failure prints the actual output in the style row format, and a diff of the expected and actual blocks (`-` is expected, `+` is actual). Check that the actual output is what the line should show before you copy it.

Run one scenario by its name (spaces become underscores), a whole feature by its tag, or everything:

```sh
go test ./features -run 'TestFeatures/Usage_of_75'   # one scenario, matched by a prefix of its name
go test ./features -feature.tags=@wrapping              # every scenario in features/wrapping.feature
go test ./features -feature.tags=@rate_bars,@wrapping   # a comma means or, && means and, ~ means not
go test ./...                                   # everything
```

When you change what the line looks like on purpose, edit the expected output in the affected scenarios and unit tests by hand and read the diff before you commit it.

Tests must never touch your real Claude Code state. The render runners point `HOME` and `CLAUDE_CONFIG_DIR` at a temporary directory, and any test you add should do the same. When you run the binary by hand, do the same:

```sh
tmp=$(mktemp -d)
echo '{"model":{"display_name":"Opus 4.7"}}' | HOME="$tmp" CLAUDE_CONFIG_DIR="$tmp" go run .
```

## Conventions

### Configuration keys

Every key the code reads must appear, with a one-line comment above it, in `internal/config/defaults.conf` (the global keys) or in the `defaults.conf` of the plugin that reads it, and every key in those files must be read by something. `TestKeysMatchDefaults` in `internal/statusline` enforces both directions, so a new setting cannot ship undocumented and a dead one cannot linger.

### Adding a built-in theme

Built-in themes are YAML files in the top-level `themes/` directory, embedded in the binary by `themes/themes.go`; the file name without `.yaml` is the name users put in `theme`. Write one in the format of the existing ones, with comments, since the files double as examples. Every theme is a complete, standalone file, so start by copying `themes/classic.yaml` or `themes/gradient.yaml`. Parsing is strict (unknown keys are an error), so a typo fails the tests rather than shipping. Add a test in `internal/theme` for what is new, and one scenario that selects the theme in its config (`features/gradient_theme.feature` is the model), so each built-in theme is applied once. Keep the loader's fallback to `classic` silent: a status line never prints an error.

### Adding a segment

A segment is a plugin: a package under `segments/` that owns its code, its `defaults.conf` and its tests, plus an entry in `segments/all.go`. A segment should be hidden, not wrong, when its data is missing.

### Go style

- Methods on a struct use pointer receivers, never value receivers, so that a type has one consistent method set.
- Every receiver is named, with a short name taken from the type (`func (m *model) Keys()`, never `func (*model) Keys()`).
- Behaviour that operates on a struct's data is a method on that struct, so it is found by looking at the type rather than by searching the package for a free function. Constructors stay functions (`New`, `Load`, `Parse`, `Open`: anything that builds and returns a value for the caller to use), and so do stateless helpers with no natural owning type, such as `payload.Sanitize`, `tokens.FormatTokens` and the colour-spec parser. Do not force a helper onto a type just to avoid a package-level function.

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
