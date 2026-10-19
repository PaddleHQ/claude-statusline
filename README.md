# claude-statusline

An open-source status line for Claude Code, written in Go. It is at an early stage: for now it shows the working directory and git branch, model, context usage, token counts, prompt-cache hit rate and rate-limit usage bars. It wraps onto further rows when the terminal is narrow, and anything unexpected gives a blank line rather than an error.

## Install

As a Claude Code plugin:

```text
/plugin marketplace add PaddleHQ/claude-statusline
/plugin install statusline@claude-statusline
```

On your next session start, a hook installs the latest release for your platform to `~/.claude/claude-statusline/bin` (under `$CLAUDE_CONFIG_DIR` if you set it). It checks for a newer release at most once a day, verifies its checksum and, when the GitHub CLI is available and logged in, its build provenance; it never downgrades and keeps the previous binary if anything fails. Without `gh`, the checksum comes from the same release as the archive, so it detects corruption but not a tampered release; the hook says so once. It then runs `claude-statusline setup`, which sets `statusLine` in `settings.json` only if you do not already have one, backing the file up first, and never touches it again.

### Manual install

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

Or build from source with Go, which installs into `$(go env GOPATH)/bin`:

```sh
go install github.com/PaddleHQ/claude-statusline@latest
```

Then add this to `~/.claude/settings.json`, using the path you installed to:

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

## Configure

Configuration is optional. Your settings live in `~/.claude/claude-statusline/config.yaml` (under `$CLAUDE_CONFIG_DIR` if you set it), as YAML:

```yaml
theme: gradient
segments: [path, model, ctx, tokens, cache, rate5h, rate7d]
```

`theme` picks the colours and bar glyphs (`classic`, `gradient` or the path to a theme file), `segments` chooses what to show and in what order, and `separator` is the glyph drawn between segments. Each segment has settings of its own, listed in the README in its folder under `segments/`, for example [model](segments/model/README.md).

## Try it

Claude Code sends the session as JSON on standard input, and the status line is what the program prints. To see it without Claude Code, pipe a session in yourself:

```sh
echo '{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"}}' | go run .
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to change the code and what CI checks.

## Licence

Released under the terms in [LICENSE](LICENSE).
