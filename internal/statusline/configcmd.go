package statusline

import (
	"errors"
	"io/fs"
	"os"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/segments"
)

// DefaultConfig is the text of the config command: every default setting, commented out. It reads no environment,
// so the same binary always prints the same text.
func DefaultConfig() string {
	return config.CommentedDefaults(segments.Defaults()...)
}

// ConfigCheck is the text of the config-check command: which config file is read and whether it can be used, then
// the settings in it that nothing reads. ok is false only when the file exists and cannot be used.
func ConfigCheck(getenv func(string) string) (report string, ok bool) {
	path := config.Path(getenv)
	if path == "" {
		return "Config file: none\nNo config directory is set, so the defaults apply.\n", true
	}
	defaults := segments.Defaults()
	cfg := config.Load(path, defaults...)

	var b strings.Builder
	b.WriteString("Config file: " + config.Clean(collapseHome(path, getenv("HOME"))) + "\n")
	ok = true
	switch p := cfg.Problem(); {
	case p != nil:
		b.WriteString(p.Report() + "\n")
		ok = false
	case missing(path):
		b.WriteString("Does not exist, so the defaults apply.\n")
	default:
		b.WriteString("Parsed without errors.\n")
	}
	if unknown := cfg.UnknownKeys(defaults...); len(unknown) > 0 {
		b.WriteString("\nUnknown settings (ignored):\n")
		for _, key := range unknown {
			b.WriteString("  " + config.Clean(key) + "\n")
		}
	}
	return b.String(), ok
}

// missing reports a file that is not there; Load treats it as an empty config, not a problem.
func missing(path string) bool {
	_, err := os.Stat(path)
	return errors.Is(err, fs.ErrNotExist)
}
