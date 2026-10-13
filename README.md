# claude-statusline

An open-source status line for Claude Code, written in Go.

It is at a very early stage: for now it prints only the model name, for example `Opus 4.7`. It reads the session JSON that Claude Code passes on stdin, prints the model's display name with no trailing newline, and always exits 0. Control characters are stripped from the name, and anything unexpected results in blank output rather than an error.

## Install

### As a Claude Code plugin

```text
/plugin marketplace add PaddleHQ/claude-statusline
/plugin install statusline@claude-statusline
```

On your next session start, a hook downloads the pinned release for your platform, checks its sha256 against the hashes committed in this repository, and installs it to `~/.claude/claude-statusline/bin`. It sets `statusLine` in `settings.json` only if you do not already have one, backing the file up first, and never touches it again. Once the right version is installed, later sessions make no network requests. Without `jq` the hook cannot edit `settings.json` and tells you what to add instead.

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
