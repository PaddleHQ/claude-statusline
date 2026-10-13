# Agent notes

Read README.md for what this repo is and how it fits together.

- Safe to run unprompted: `go test ./...`, `golangci-lint run`, `goreleaser check`, `goreleaser release --snapshot --clean --skip=publish`, `sh tests/bootstrap_test.sh`.
- Ask first before anything that creates a release or tag, pushes, or edits `~/.claude/settings.json`. Run the binary by hand only with `HOME` and `CLAUDE_CONFIG_DIR` pointing at a temporary directory.

@CONTRIBUTING.md
