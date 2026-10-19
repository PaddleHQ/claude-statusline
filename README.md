# claude-statusline

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/images/hero-dark.svg">
  <img alt="The status line on one row: working directory and git branch, model and effort, a context usage bar, token counts, prompt cache hit rate, boot and skill sizes, and 5 hour and 7 day rate-limit bars with pace markers" src="docs/images/hero-light.svg">
</picture>

An open-source status line for Claude Code, written in Go. It is at an early stage: for now it shows the working directory and git branch, model, context usage, token counts, prompt-cache hit rate and rate-limit usage bars. It wraps onto further rows when the terminal is narrow, and anything unexpected gives a blank line rather than an error.

## Install

As a Claude Code plugin:

```text
/plugin marketplace add PaddleHQ/claude-statusline
/plugin install statusline@claude-statusline
```

On your next session start, a hook installs the latest release for your platform, checks it, and sets up the status line if you do not already have one. To install by hand from a release or with Go, see [Manual install](docs/how_it_works.md#manual-install).

## Configure

Configuration is optional. Your settings live in `~/.claude/claude-statusline/config.yaml` (under `$CLAUDE_CONFIG_DIR` if you set it), as YAML:

```yaml
theme: gradient
segments: [path, model, ctx, tokens, cache, rate5h, rate7d]
```

With `theme: gradient`, the bars are shaded from green to red along their length:

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/images/gradient-dark.svg">
  <img alt="The status line with the gradient theme: the context and rate-limit bars are drawn as segmented cells shaded from green towards red, with a slash marking the pace point" src="docs/images/gradient-light.svg">
</picture>

`theme` picks the colours and bar glyphs, and `segments` chooses what to show and in what order. Every key, custom segments, wrapping and themes are in [Customising](docs/customising.md). To see what your own line shows, run `/statusline:help` in Claude Code, or `claude-statusline help` if you installed by hand.

## More

- [How it works](docs/how_it_works.md): what runs when, the shared rate-limit cache, how the plugin updates itself, and manual install.
- [Customising](docs/customising.md): every setting, custom segments, subagent rows and themes.
- [CONTRIBUTING.md](CONTRIBUTING.md): how to change the code.

## Licence

Released under the terms in [LICENSE](LICENSE).
