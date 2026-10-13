# claude-statusline

An open-source status line for Claude Code, written in Go.

It is at an early stage: for now it prints the model, effort level and rate-limit usage bars, for example `Opus 4.7 [high]`. It reads the session JSON that Claude Code passes on stdin, prints one line with no trailing newline, and always exits 0. Control characters are stripped from every string taken from the session, and anything unexpected results in blank output rather than an error.

## Install

### As a Claude Code plugin

```text
/plugin marketplace add PaddleHQ/claude-statusline
/plugin install statusline@claude-statusline
```

On your next session start, a hook downloads the pinned release for your platform, checks its sha256 against the hashes committed in this repository, and installs it to `~/.claude/claude-statusline/bin`. It then runs `claude-statusline setup`, which sets `statusLine` in `settings.json` only if you do not already have one, backing the file up first, and never touches it again. Once the right version is installed, later sessions make no network requests.

### Manual install

#### From a release

Each [release](https://github.com/PaddleHQ/claude-statusline/releases) has a binary for macOS and Linux on arm64 and amd64, plus `checksums.txt`, an SBOM per archive and a GitHub build provenance attestation. Set the version and your platform, then download, verify and install:

```sh
VERSION=0.1.0   # without the leading v
OS=darwin       # or linux
ARCH=arm64      # or amd64
ARCHIVE="claude-statusline_${VERSION}_${OS}_${ARCH}.tar.gz"

gh release download "v${VERSION}" --repo PaddleHQ/claude-statusline \
  --pattern "$ARCHIVE" --pattern checksums.txt
shasum -a 256 -c checksums.txt --ignore-missing
gh attestation verify "$ARCHIVE" --repo PaddleHQ/claude-statusline

mkdir -p ~/.local/bin
tar -xzf "$ARCHIVE" -C ~/.local/bin claude-statusline
```

`shasum` checks the archive against the published checksums. `gh attestation verify` proves it was built by this repository's release workflow.

#### With Go

```sh
go install github.com/PaddleHQ/claude-statusline@latest
```

This builds from source into `$(go env GOPATH)/bin`.

#### Configure Claude Code

Add this to `~/.claude/settings.json`, using the path you installed to:

```json
{
  "statusLine": {
    "type": "command",
    "command": "~/.local/bin/claude-statusline",
    "refreshInterval": 60
  }
}
```

Run `claude-statusline --version` to print the installed version.

## Configuration

Configuration is optional. Your settings live in `~/.claude/claude-statusline/config`, or under `$CLAUDE_CONFIG_DIR` if you set it. Each line is `key = value`. Blank lines and full-line `#` comments are ignored, whitespace around keys and values is trimmed, and the last occurrence of a key wins. Keys are lower case, with at most one dot (`model.effort`); anything else is ignored. The file is parsed, never executed, so values are always plain text. A missing or invalid value falls back to its default.

[`internal/config/defaults.conf`](internal/config/defaults.conf) documents every key with a comment and its default value. The current keys are:

| Key | Default | Meaning |
|---|---|---|
| `segments` | `model rate5h rate7d` | Segments to show, in order. Unknown names are skipped. |
| `separator` | `·` | Glyph drawn between segments. |
| `theme` | `default` | Colour theme. Unknown names fall back to `default`. |
| `model.effort` | `true` | Show the effort level after the model name (`true` or `false`). |
| `rate5h.label`, `rate7d.label` | `5h`, `7d` | Text shown before the bar. |
| `rate5h.width`, `rate7d.width` | `20` | Bar width in cells on a wide terminal. |
| `rate5h.warn`, `rate7d.warn` | `20`, `10` | Pace deviation, in percentage points, beyond which the fill turns amber. |
| `rate5h.bad`, `rate7d.bad` | `40`, `20` | Pace deviation, in percentage points, beyond which the fill turns red. |
| `rate5h.hard_cap`, `rate7d.hard_cap` | `95` | Usage percentage at or above which the fill is always red. |

Set `NO_COLOR` to any non-empty value to turn colour and styling off.

### Rate-limit bars

The `rate5h` and `rate7d` segments draw your 5 hour and 7 day usage as a bar followed by the percentage. The vertical marker `┃` sits at the point in the window you would reach if you spent your quota evenly. The fill is green while your usage is close to the marker, amber once it drifts more than `warn` points either way, and red beyond `bad` points or at `hard_cap`. A bar disappears once its window has reset. Bars narrow on small terminals: half width below 120 columns and 70% below 160 (`$COLUMNS`), never under 4 cells.

Rate limits belong to your account, not to a session, and an idle session's payload goes stale. Readings are therefore shared between sessions through `~/.claude/claude-statusline/cache/ratelimits` (under `$CLAUDE_CONFIG_DIR` if set). A session replaces the cached reading only with a newer window, higher usage, or a drop of more than 10 points (a mid-window quota reset), and the line is always drawn from the merged value. The file is private state and may change between versions.

## Development

```sh
go test ./...
golangci-lint run
```

Shell scripts are checked with `shellcheck -s sh hooks/bootstrap.sh scripts/pin-release.sh tests/bootstrap_test.sh` and tested with `sh tests/bootstrap_test.sh`.

Every pull request needs exactly one of the labels `norelease`, `release:major`, `release:minor` or `release:patch`. Merging a `release:*` pull request tags the next semantic version and publishes the release binaries.

### Pinning a release for the plugin

The plugin installs the version recorded in `release/version`, not the latest release. After a release is published, run `scripts/pin-release.sh <version>` (for example `0.1.0`, without the leading `v`). It needs `gh`, verifies the build attestation and checksum of all four archives, and only then writes `release/version` and `release/checksums.txt`. Commit both files in a pull request. Until it merges, the plugin keeps installing the previous pin. While `release/version` is empty, the hook does nothing.

## Licence

See [LICENSE](LICENSE).
