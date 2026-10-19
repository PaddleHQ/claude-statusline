# claude-statusline

An open-source status line for Claude Code, written in Go. It is at an early stage: for now it shows the model and its effort level, for example `Opus 4.7 [high]`. Anything unexpected gives a blank line rather than an error.

## Configure

Configuration is optional. Your settings live in `~/.claude/claude-statusline/config.yaml` (under `$CLAUDE_CONFIG_DIR` if you set it), as YAML:

```yaml
theme: gradient
segments: [model]
```

`theme` picks the colours (`classic`, `gradient` or the path to a theme file), `segments` chooses what to show and in what order, and `separator` is the glyph drawn between segments. Each segment has settings of its own, listed in the README in its folder under `segments/`, for example [model](segments/model/README.md).

## Try it

Claude Code sends the session as JSON on standard input, and the status line is what the program prints. To see it without Claude Code, pipe a session in yourself:

```sh
echo '{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"}}' | go run .
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to change the code and what CI checks.

## Licence

Released under the terms in [LICENSE](LICENSE).
