package statusline

import (
	"fmt"
	"slices"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/hook"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/segments"
)

const docsURL = "https://github.com/PaddleHQ/claude-statusline/blob/main/docs/customising.md"

// helper describes the status line as the user has configured it. It never runs a hook script.
type helper struct {
	env        *segments.Env
	hooks      []hook.Hook
	names      []string
	excluded   []string
	configPath string
	problem    *config.Problem
}

// Help describes the user's status line in plain text: what is shown now, the custom segments and the built-ins left out.
// It reads the same config and hook directory as Build, and names the config file with why it cannot be used, if it cannot.
func Help(getenv func(string) string) string {
	defaults := segments.Defaults()
	cfg := config.New("", defaults...)
	path := config.Path(getenv)
	if path != "" {
		cfg = config.Load(path, defaults...)
	}
	th := theme.Load(cfg.String(keyTheme), getenv("HOME"), config.AppDir(getenv))
	th = th.WithoutColor()
	env := segments.NewEnv(segments.Env{Config: cfg, Theme: th, Home: getenv("HOME"), Columns: defaultColumns})

	hooks := hook.Discover(hookDir(getenv))
	excluded := excludedNames(cfg, hooks)
	h := &helper{
		env:        env,
		hooks:      hooks,
		names:      segmentNames(cfg, excluded),
		excluded:   excluded,
		configPath: path,
		problem:    cfg.Problem(),
	}
	return h.text()
}

func (h *helper) text() string {
	var b strings.Builder
	b.WriteString("Your status line, as currently configured.\n\n")

	b.WriteString("Shown now, in order:\n")
	shown := 0
	for _, name := range h.names {
		if lines := h.shown(name); len(lines) > 0 {
			shown += len(lines)
			writeList(&b, lines)
		}
	}
	if shown == 0 {
		b.WriteString("  (nothing: your segments list names no known segment)\n")
	}

	if custom := h.custom(); len(custom) > 0 {
		b.WriteString("\nCustom segments:\n")
		writeList(&b, custom)
	}

	if left := h.unlisted(); len(left) > 0 {
		b.WriteString("\nAlso available, not in your segments list:\n")
		writeList(&b, left)
	}

	if excluded := h.excludedLines(); len(excluded) > 0 {
		b.WriteString("\nLeft out by segments_excluded:\n")
		writeList(&b, excluded)
	}

	b.WriteString("\n" + h.configFooter())
	b.WriteString("Documentation: " + docsURL + "\n")
	return b.String()
}

// shown describes what a segment name stands for: the hooks of that name, which replace a built-in, else the built-in.
func (h *helper) shown(name string) []string {
	var lines []string
	for i := range h.hooks {
		if h.hooks[i].Name == name {
			lines = append(lines, name+": "+hookHelp(&h.hooks[i]))
		}
	}
	if len(lines) > 0 {
		return lines
	}
	if seg, ok := h.builtin(name); ok {
		return []string{name + ": " + seg.Help()}
	}
	return nil
}

func (h *helper) custom() []string {
	lines := make([]string, 0, len(h.hooks))
	for i := range h.hooks {
		if slices.Contains(h.excluded, h.hooks[i].Name) {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s (%s, slot %s): %s", h.hooks[i].Name, h.hooks[i].File, h.hooks[i].Slot, hookHelp(&h.hooks[i])))
	}
	return lines
}

func (h *helper) unlisted() []string {
	listed := map[string]bool{}
	for _, name := range h.names {
		listed[name] = true
	}
	var lines []string
	for _, seg := range h.env.All() {
		if !listed[seg.Name()] && !slices.Contains(h.excluded, seg.Name()) {
			lines = append(lines, seg.Name()+": "+seg.Help())
		}
	}
	return lines
}

// excludedLines describes each excluded name that stands for a hook or a built-in; other names describe nothing.
func (h *helper) excludedLines() []string {
	lines := make([]string, 0, len(h.excluded))
	for _, name := range h.excluded {
		lines = append(lines, h.shown(name)...)
	}
	return lines
}

func (h *helper) builtin(name string) (segments.Segment, bool) {
	seg, _, ok := h.env.Lookup(name)
	return seg, ok
}

func hookHelp(h *hook.Hook) string {
	if h.Help != "" {
		return h.Help
	}
	return "custom segment from " + h.File
}

func writeList(b *strings.Builder, lines []string) {
	for _, l := range lines {
		b.WriteString("  " + config.Clean(l) + "\n")
	}
}

func (h *helper) configFooter() string {
	path := config.Clean(h.configPath)
	switch {
	case h.configPath == "":
		return "Config file: none (no Claude config directory found), so the defaults apply.\n"
	case h.problem != nil:
		return "Config file: " + path + "\nConfig error: " + path + h.problemText() + "\n"
	case missing(h.configPath):
		return fmt.Sprintf("Config file: %s (not created yet, so the defaults apply)\n", path)
	default:
		return "Config file: " + path + "\n"
	}
}

// problemText follows the path: the line, when the parser gave one, and the reason.
func (h *helper) problemText() string {
	return lineSuffix(h.problem.Line) + config.Clean(h.problem.Reason)
}
