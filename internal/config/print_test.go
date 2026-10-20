package config_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/segments"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const printHeader = "# # claude-statusline settings, all commented out so the defaults still apply."

func uncomment(t *testing.T, text string) string {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	for i, line := range lines {
		switch {
		case line == "#":
			lines[i] = ""
		case strings.HasPrefix(line, "# "):
			lines[i] = line[2:]
		default:
			require.Failf(t, "line is not commented", "%q", line)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func flatKeys(parsed map[string]any) []string {
	var keys []string
	for k, v := range parsed {
		if section, ok := v.(map[string]any); ok {
			for name := range section {
				keys = append(keys, k+"."+name)
			}
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func TestCommentedDefaultsRoundTrip(t *testing.T) {
	plugins := segments.Defaults()
	stripped := uncomment(t, config.CommentedDefaults(plugins...))

	got := config.Parse(stripped)
	require.NotNil(t, got)
	want := config.Parse(strings.Join(append([]string{config.Defaults}, plugins...), "\n"))
	require.NotNil(t, want)
	assert.Equal(t, want, got)

	wantKeys := config.DefaultKeys(plugins...)
	sort.Strings(wantKeys)
	assert.Equal(t, wantKeys, flatKeys(got))
}

func TestCommentedDefaultsAllLinesCommented(t *testing.T) {
	out := config.CommentedDefaults(segments.Defaults()...)
	for line := range strings.SplitSeq(strings.TrimSuffix(out, "\n"), "\n") {
		assert.True(t, strings.HasPrefix(line, "#"), "line %q", line)
	}
	assert.Nil(t, config.Parse(out))
}

func TestCommentedDefaultsLayout(t *testing.T) {
	out := config.CommentedDefaults(segments.Defaults()...)
	assert.True(t, strings.HasSuffix(out, "\n"))
	assert.False(t, strings.HasSuffix(out, "\n\n"))
	for line := range strings.SplitSeq(strings.TrimSuffix(out, "\n"), "\n") {
		assert.Equal(t, strings.TrimRight(line, " \t"), line)
	}
	assert.Equal(t, printHeader, strings.SplitN(out, "\n", 2)[0])
}

func TestCommentedDefaultsTransform(t *testing.T) {
	tests := []struct {
		name    string
		plugins []string
		want    string
	}{
		{"blank line", []string{"a: 1\n\nb: 2\n"}, "# a: 1\n#\n# b: 2\n"},
		{"indent kept", []string{"s:\n  key: 1\n"}, "# s:\n#   key: 1\n"},
		{"hash in value", []string{"separator: \"#\"\n"}, "# separator: \"#\"\n"},
		{"no final newline", []string{"a: 1", "b: 2"}, "# a: 1\n#\n# b: 2\n"},
		{"crlf", []string{"a: 1\r\nb: 2\r\n"}, "# a: 1\n# b: 2\n"},
		{"trailing blank lines", []string{"a: 1\n\n\n", "b: 2\n"}, "# a: 1\n#\n# b: 2\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := config.CommentedDefaults(tt.plugins...)
			assert.True(t, strings.HasSuffix(out, "\n"+tt.want), "%q", out)
		})
	}
}

func TestCommentedDefaultsIsDeterministic(t *testing.T) {
	plugins := segments.Defaults()
	first := config.CommentedDefaults(plugins...)
	second := config.CommentedDefaults(plugins...)
	assert.Equal(t, first, second)
}
