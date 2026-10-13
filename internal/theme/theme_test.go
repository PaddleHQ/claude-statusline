package theme_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/PaddleHQ/claude-statusline/internal/theme"
)

func TestLookupDefault(t *testing.T) {
	t.Parallel()

	got := theme.Lookup("default")
	want := theme.Theme{
		Muted:       "\033[2m",
		Good:        "\033[32m",
		Warn:        "\033[33m",
		Bad:         "\033[31m",
		Marker:      "\033[1m",
		Reset:       "\033[0m",
		Full:        "█",
		Partials:    [7]string{"▏", "▎", "▍", "▌", "▋", "▊", "▉"},
		Empty:       "░",
		MarkerGlyph: "┃",
	}
	assert.Equal(t, want, got)
}

func TestLookupUnknownFallsBack(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"", "nonexistent", "Default", "../x"} {
		assert.Equal(t, theme.Lookup("default"), theme.Lookup(name), "theme %q", name)
	}
}

func TestWithoutColor(t *testing.T) {
	t.Parallel()

	d := theme.Lookup("default")
	got := d.WithoutColor()
	for _, colour := range []string{got.Text, got.Muted, got.Good, got.Warn, got.Bad, got.Marker, got.Reset} {
		assert.Empty(t, colour, "WithoutColor() kept a colour: %+v", got)
	}
	assert.Equal(t, d.Full, got.Full)
	assert.Equal(t, d.Partials, got.Partials)
	assert.Equal(t, d.Empty, got.Empty)
	assert.Equal(t, d.MarkerGlyph, got.MarkerGlyph)
	assert.NotEmpty(t, d.Muted, "WithoutColor() modified the receiver")
}

func TestPaint(t *testing.T) {
	t.Parallel()

	d := theme.Lookup("default")
	tests := []struct {
		name   string
		colour string
		want   string
	}{
		{"wraps with colour and reset", d.Muted, "\033[2mx\033[0m"},
		{"leaves text bare without colour", d.Text, "x"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, d.Paint(tt.colour, "x"), tt.name)
	}
}
