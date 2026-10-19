package statusline_test

import (
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/statusline"
	"github.com/PaddleHQ/claude-statusline/segments"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	got := statusline.DefaultConfig()

	assert.Equal(t, config.CommentedDefaults(segments.Defaults()...), got)

	last := -1
	for _, section := range segments.Defaults() {
		first, _, _ := strings.Cut(strings.TrimSpace(section), "\n")
		for line := range strings.SplitSeq(section, "\n") {
			if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, " ") {
				first = line
				break
			}
		}
		at := strings.Index(got, "# "+first)
		if at < 0 {
			continue
		}
		assert.Greater(t, at, last, "section starting %q is out of order", first)
		last = at
	}
}
