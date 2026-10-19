package statusline

import (
	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/segments"
)

// DefaultConfig is the text of the config command: every default setting, commented out. It reads no environment,
// so the same binary always prints the same text.
func DefaultConfig() string {
	return config.CommentedDefaults(segments.Defaults()...)
}
