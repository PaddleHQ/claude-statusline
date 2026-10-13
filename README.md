# claude-statusline

An open-source status line for Claude Code, written in Go.

It is at an early stage: for now it prints the model name and effort level, for example `Opus 4.7 [high]`. It reads the session JSON that Claude Code passes on stdin, prints one line with no trailing newline, and always exits 0. Control characters are stripped from every string taken from the session, and anything unexpected results in blank output rather than an error.

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
| `segments` | `model` | Segments to show, in order. Unknown names are skipped. |
| `separator` | `·` | Glyph drawn between segments. |
| `theme` | `default` | Colour theme. Unknown names fall back to `default`. |
| `model.effort` | `true` | Show the effort level after the model name (`true` or `false`). |

Set `NO_COLOR` to any non-empty value to turn colour and styling off.

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
