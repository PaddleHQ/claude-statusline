package statusline

import (
	"fmt"
	"os"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/hook"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/segments"
)

const docsURL = "https://github.com/PaddleHQ/claude-statusline/blob/main/docs/customising.md"

// helper describes the status line as the user has configured it. It never runs a hook script.
type helper struct {
	env   *segments.Env
	hooks []hook.Hook
	names []string
}

// Help describes the user's status line in plain text: what is shown now, the custom segments and the built-ins left out.
// It reads the same config and hook directory as Build, and never reports a failure.
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

	h := &helper{
		env:   env,
		hooks: hook.Discover(hookDir(getenv)),
		names: strings.Fields(cfg.String("segments")),
	}
	return h.text(path)
}

func (h *helper) text(configPath string) string {
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

	if len(h.hooks) > 0 {
		b.WriteString("\nCustom segments:\n")
		writeList(&b, h.custom())
	}

	if left := h.unlisted(); len(left) > 0 {
		b.WriteString("\nAlso available, not in your segments list:\n")
		writeList(&b, left)
	}

	b.WriteString("\n" + configFooter(configPath))
	b.WriteString("Documentation: " + docsURL + "\n")
	return b.String()
}

// shown describes what the name stands for in the segments list: the hooks of that name, which replace a built-in, else the built-in.
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
		if !listed[seg.Name()] {
			lines = append(lines, seg.Name()+": "+seg.Help())
		}
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
		b.WriteString("  " + l + "\n")
	}
}

func configFooter(path string) string {
	switch {
	case path == "":
		return "Config file: none (no Claude config directory found), so the defaults apply.\n"
	case fileExists(path):
		return "Config file: " + path + "\n"
	default:
		return fmt.Sprintf("Config file: %s (not created yet, so the defaults apply)\n", path)
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
