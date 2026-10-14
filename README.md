# claude-statusline

An open-source status line for Claude Code, written in Go. It is at an early stage: for now it shows the model and its effort level, for example `Opus 4.7 [high]`, and anything unexpected gives a blank line rather than an error.

## Try it

Claude Code sends the session as JSON on standard input, and the status line is what the program prints. To see it without Claude Code, pipe a session in yourself:

```sh
echo '{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"}}' | go run .
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to change the code and what CI checks.

## Licence

Released under the terms in [LICENSE](LICENSE).
