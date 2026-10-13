# claude-statusline

An open-source status line for Claude Code, written in Go.

It is at a very early stage: for now it prints only the model name, for example `Opus 4.7`. It reads the session JSON that Claude Code passes on stdin, prints the model's display name with no trailing newline, and always exits 0. Control characters are stripped from the name, and anything unexpected results in blank output rather than an error.

## Install

```sh
go install github.com/PaddleHQ/claude-statusline@latest
```

Then add it to `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "claude-statusline",
    "refreshInterval": 60
  }
}
```

`$(go env GOPATH)/bin` must be on your `PATH`, otherwise use the full path to the binary as the command.

## Development

```sh
go test ./...
golangci-lint run
```

## Licence

See [LICENSE](LICENSE).
