package bar_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/PaddleHQ/claude-statusline/internal/bar"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
)

func TestWidthFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		columns int
		want    int
	}{
		{"wide terminal", 200, 20},
		{"exactly 160 is wide", 160, 20},
		{"159 is medium", 159, 14},
		{"exactly 120 is medium", 120, 14},
		{"119 is narrow", 119, 10},
		{"tiny", 10, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, bar.WidthFor(tt.columns, 20, 14, 10))
		})
	}

	assert.Equal(t, 4, bar.WidthFor(10, 20, 14, 2), "floor of four")
	assert.Equal(t, 4, bar.WidthFor(999, 1, 1, 1), "floor applies on wide terminals")
	assert.Equal(t, 200, bar.WidthFor(999, 5000, 5000, 5000), "cap of 200")
}

func TestElapsedPercent(t *testing.T) {
	t.Parallel()

	const window = 18000
	tests := []struct {
		name          string
		now, resetsAt int64
		want          int
	}{
		{"halfway", 1000, 1000 + window/2, 50},
		{"just started", 1000, 1000 + window, 0},
		{"before the window", 0, 1000 + window, 0},
		{"after the window is clamped", 5000, 1000, 100},
		{"rounds down", 1000, 1000 + window - 179, 0},
		{"one percent", 1000, 1000 + window - 180, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, bar.ElapsedPercent(tt.now, tt.resetsAt, window))
		})
	}
}

// markers uses plain letters so cells can be counted without escape codes.
var markers = theme.Theme{
	Good: "<g>", Warn: "<w>", Bad: "<b>", Marker: "<m>", Muted: "<d>", Reset: "</>",
	Full: "F", Partials: [7]string{"1", "2", "3", "4", "5", "6", "7"}, Empty: "E", MarkerGlyph: "M",
}

func TestRender(t *testing.T) {
	t.Parallel()

	plain := markers.WithoutColor()
	tests := []struct {
		name string
		p    bar.Params
		th   theme.Theme
		want string
	}{
		{"empty bar with marker at the start", bar.Params{Usage: 0, Target: 0, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, plain, "MEEE"},
		{"half full, marker inside the fill", bar.Params{Usage: 50, Target: 50, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, plain, "FFME"},
		{"partial cell uses the matching glyph", bar.Params{Usage: 10, Target: 90, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, plain, "3EEM"},
		{"rounds eighths to nearest", bar.Params{Usage: 45, Target: 90, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, plain, "F6EM"},
		{"full bar", bar.Params{Usage: 100, Target: 0, Width: 3, Warn: 20, Bad: 40, HardCap: 95}, plain, "MFF"},
		{"usage over 100 is clamped", bar.Params{Usage: 400, Target: 0, Width: 3, Warn: 20, Bad: 40, HardCap: 500}, plain, "MFF"},
		{"target at 100 sits in the last cell", bar.Params{Usage: 0, Target: 100, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, plain, "EEEM"},
		{"good within warn", bar.Params{Usage: 70, Target: 50, Width: 2, Warn: 20, Bad: 40, HardCap: 95}, markers, "<g>F</><m>M</>"},
		{"warn just past the limit", bar.Params{Usage: 71, Target: 50, Width: 2, Warn: 20, Bad: 40, HardCap: 95}, markers, "<w>F</><m>M</>"},
		{"bad just past the limit", bar.Params{Usage: 91, Target: 50, Width: 2, Warn: 20, Bad: 40, HardCap: 95}, markers, "<b>F</><m>M</>"},
		{"behind pace also deviates", bar.Params{Usage: 5, Target: 50, Width: 2, Warn: 20, Bad: 40, HardCap: 95}, markers, "<b>1</><m>M</>"},
		{"hard cap overrides pace", bar.Params{Usage: 96, Target: 96, Width: 2, Warn: 20, Bad: 40, HardCap: 95}, markers, "<b>F</><m>M</>"},
		{"empty cells are muted", bar.Params{Usage: 0, Target: 0, Width: 2, Warn: 20, Bad: 40, HardCap: 95}, markers, "<m>M</><d>E</>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, bar.Render(tt.p, tt.th))
		})
	}
}

func TestRenderWidth(t *testing.T) {
	t.Parallel()

	got := bar.Render(bar.Params{Usage: 33, Target: 66, Width: 20, Warn: 20, Bad: 40, HardCap: 95}, markers.WithoutColor())
	assert.Len(t, []rune(got), 20, "bar = %q", got)
	assert.Equal(t, 1, strings.Count(got, "M"), "bar should hold exactly one marker: %q", got)
}

func TestRenderNoTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		usage int
		want  string
	}{
		{"below warn is good", 49, "<g>F</><g>F</><d>E</><d>E</>"},
		{"at warn is amber", 50, "<w>F</><w>F</><d>E</><d>E</>"},
		{"just below bad is amber", 89, "<w>F</><w>F</><w>F</><w>4</>"},
		{"at bad is red", 90, "<b>F</><b>F</><b>F</><b>5</>"},
		{"empty has no marker", 0, "<d>E</><d>E</><d>E</><d>E</>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := bar.Render(bar.Params{Usage: tt.usage, Target: 10, Width: 4, Warn: 50, Bad: 90, HardCap: 1, NoTarget: true}, markers)
			assert.Equal(t, tt.want, got)
			assert.NotContains(t, got, "M")
		})
	}
}
