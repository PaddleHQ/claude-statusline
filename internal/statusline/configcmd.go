package statusline

import (
	"errors"
	"fmt"
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
	cfg := config.Resolve(getenv, defaults...)

	opts := cfg.PluginOptions()
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
	writePluginOptions(&b, cfg, collapseHome(opts.Path, getenv("HOME")))
	return b.String(), ok
}

// writePluginOptions reports the plugin options in settings.json: the file, problems with it, and where each
// setting they can supply ended up coming from. It writes nothing when the file has no options and no problems.
func writePluginOptions(b *strings.Builder, cfg *config.Config, shown string) {
	opts := cfg.PluginOptions()
	if !opts.Active() && len(opts.Issues) == 0 {
		return
	}
	fmt.Fprintf(b, "\nPlugin options: %s", config.Clean(shown))
	if opts.Active() {
		fmt.Fprintf(b, " (%s)", config.Clean(opts.Entry))
	}
	b.WriteString("\n")
	for _, issue := range opts.Issues {
		fmt.Fprintf(b, "  %s\n", config.Clean(issue))
	}
	if !opts.Active() {
		b.WriteString("  Ignored, so the config file and the defaults apply.\n")
		return
	}
	for _, o := range config.Options() {
		fmt.Fprintf(b, "  %s: %s\n", o.Key, cfg.Source(o.Key))
	}
}

// missing reports a file that is not there; Load treats it as an empty config, not a problem.
func missing(path string) bool {
	_, err := os.Stat(path)
	return errors.Is(err, fs.ErrNotExist)
}
