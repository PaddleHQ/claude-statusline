package bar

import (
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/theme"
)

func TestResponsiveWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		width, columns int
		want           int
	}{
		{"wide terminal keeps the width", 20, 200, 20},
		{"exactly 160 keeps the width", 20, 160, 20},
		{"medium terminal takes 70%", 20, 140, 14},
		{"exactly 120 is medium", 20, 120, 14},
		{"narrow terminal halves", 20, 100, 10},
		{"floor of four", 6, 10, 4},
		{"floor applies on wide terminals", 2, 999, 4},
		{"cap of 200", 5000, 999, 200},
		{"rounds down", 15, 140, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ResponsiveWidth(tt.width, tt.columns); got != tt.want {
				t.Errorf("ResponsiveWidth(%d, %d) = %d, want %d", tt.width, tt.columns, got, tt.want)
			}
		})
	}
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
			if got := ElapsedPercent(tt.now, tt.resetsAt, window); got != tt.want {
				t.Errorf("ElapsedPercent() = %d, want %d", got, tt.want)
			}
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
		p    Params
		th   theme.Theme
		want string
	}{
		{"empty bar with marker at the start", Params{Usage: 0, Target: 0, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, plain, "MEEE"},
		{"half full, marker inside the fill", Params{Usage: 50, Target: 50, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, plain, "FFME"},
		{"partial cell uses the matching glyph", Params{Usage: 10, Target: 90, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, plain, "3EEM"},
		{"rounds eighths to nearest", Params{Usage: 45, Target: 90, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, plain, "F6EM"},
		{"full bar", Params{Usage: 100, Target: 0, Width: 3, Warn: 20, Bad: 40, HardCap: 95}, plain, "MFF"},
		{"usage over 100 is clamped", Params{Usage: 400, Target: 0, Width: 3, Warn: 20, Bad: 40, HardCap: 500}, plain, "MFF"},
		{"target at 100 sits in the last cell", Params{Usage: 0, Target: 100, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, plain, "EEEM"},
		{"good within warn", Params{Usage: 70, Target: 50, Width: 2, Warn: 20, Bad: 40, HardCap: 95}, markers, "<g>F</><m>M</>"},
		{"warn just past the limit", Params{Usage: 71, Target: 50, Width: 2, Warn: 20, Bad: 40, HardCap: 95}, markers, "<w>F</><m>M</>"},
		{"bad just past the limit", Params{Usage: 91, Target: 50, Width: 2, Warn: 20, Bad: 40, HardCap: 95}, markers, "<b>F</><m>M</>"},
		{"behind pace also deviates", Params{Usage: 5, Target: 50, Width: 2, Warn: 20, Bad: 40, HardCap: 95}, markers, "<b>1</><m>M</>"},
		{"hard cap overrides pace", Params{Usage: 96, Target: 96, Width: 2, Warn: 20, Bad: 40, HardCap: 95}, markers, "<b>F</><m>M</>"},
		{"empty cells are muted", Params{Usage: 0, Target: 0, Width: 2, Warn: 20, Bad: 40, HardCap: 95}, markers, "<m>M</><d>E</>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Render(tt.p, tt.th); got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderWidth(t *testing.T) {
	t.Parallel()

	got := Render(Params{Usage: 33, Target: 66, Width: 20, Warn: 20, Bad: 40, HardCap: 95}, markers.WithoutColor())
	if n := len([]rune(got)); n != 20 {
		t.Errorf("bar has %d cells, want 20: %q", n, got)
	}
	if strings.Count(got, "M") != 1 {
		t.Errorf("bar should hold exactly one marker: %q", got)
	}
}
