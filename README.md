# claude-statusline

An open-source status line for Claude Code, written in Go.

It is at a very early stage: for now it prints only the model name, for example `Opus 4.7`. It reads the session JSON that Claude Code passes on stdin, prints the model's display name with no trailing newline, and always exits 0. Control characters are stripped from the name, and anything unexpected results in blank output rather than an error.

## Install

### From a release

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

### With Go

```sh
go install github.com/PaddleHQ/claude-statusline@latest
```

This builds from source into `$(go env GOPATH)/bin`.

### Configure Claude Code

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

Every pull request needs exactly one of the labels `norelease`, `release:major`, `release:minor` or `release:patch`. Merging a `release:*` pull request tags the next semantic version and publishes the release binaries.

## Licence

See [LICENSE](LICENSE).
