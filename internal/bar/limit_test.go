package bar_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/bar"
	"github.com/PaddleHQ/claude-statusline/internal/theme"

	"github.com/stretchr/testify/assert"
)

func TestRenderLimit(t *testing.T) {
	t.Parallel()

	limited := func(usage, limit, width int, reached bool) bar.Params {
		return bar.Params{Usage: usage, Width: width, Warn: 50, Bad: 90, Limited: true, Limit: limit, Reached: reached}
	}
	tests := []struct {
		name string
		p    bar.Params
		th   theme.Theme
		want string
	}{
		{"marker in the empty part, good", limited(30, 80, 10, false), markers, "<g>FFF</><d>EEEEE</><m>M</><d>E</>"},
		{"marker inside the fill when reached", limited(85, 80, 10, true), markers, "<b>FFFFFFFF</><m>M</><d>E</>"},
		{"reached is bad below the bad threshold", limited(30, 80, 4, true), markers, "<b>F2</><d>E</><m>M</>"},
		{"warn still applies below the limit", limited(60, 80, 4, false), markers, "<w>FF3</><m>M</>"},
		{"a limit of 100 draws no marker", limited(30, 100, 4, false), markers, "<g>F2</><d>EE</>"},
		{"a negative limit draws no marker", limited(30, -1, 4, false), markers, "<g>F2</><d>EE</>"},
		{"reached without a marker is still bad", limited(30, -1, 4, true), markers, "<b>F2</><d>EE</>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.Render(tt.th))
		})
	}
}

func TestRenderLimitOnEveryStyle(t *testing.T) {
	t.Parallel()

	gradient := theme.Theme{
		Good: "<g>", WarnColor: "<w>", BadColor: "<b>", Reset: "</>",
		Full: "F", Empty: "E", MarkerGlyph: "X",
		Gradient: true, Ramp: [][3]int{{0, 0, 0}, {100, 100, 100}},
		EmptyColor: "<e>", MarkerColor: "<m>",
	}
	plainBlade := blade
	plainBlade.Glow = 0
	p := bar.Params{Usage: 30, Width: 10, Warn: 50, Bad: 90, Limited: true, Limit: 80, Reached: true}

	tests := []struct {
		name       string
		th         theme.Theme
		wantMarker string
		wantFill   string
	}{
		{"solid", markers, "<m>M</>", "<b>"},
		{"gradient", gradient, "<m>X</>", "<b>"},
		{"saber", plainBlade, "<m>M</>", plainBlade.BladeColor(theme.StateBad, 0, 3)},
		{"capsule", capsule, "<m>\uee04</>", "<b>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, p.Render(tt.th), tt.wantMarker)
			// A gradient bar colors its cells by position, so reached shows on the percentage instead.
			assert.Contains(t, p.Render(tt.th)+p.PercentColor(tt.th), tt.wantFill)
		})
	}
}

func TestMeterLimitDrawsTheMarkerAndPercentage(t *testing.T) {
	t.Parallel()

	m := &bar.Meter{Theme: markers, Width: 4}
	assert.Equal(t, "<g>F</><d>EE</><m>M</> 25%", m.Limit(25, 80, false, 50, 90))
	assert.Equal(t, "<b>F</><d>EE</><m>M</> 25%", m.Limit(25, 80, true, 50, 90), "reached colors the fill bad")
}

func TestFillOfALimitedBar(t *testing.T) {
	t.Parallel()

	classic := theme.Load("classic", "", "")
	tests := []struct {
		name string
		p    bar.Params
		want string
	}{
		{"good", bar.Params{Usage: 30, Limited: true, Limit: 80, Warn: 50, Bad: 90}, classic.Good},
		{"warn", bar.Params{Usage: 60, Limited: true, Limit: 80, Warn: 50, Bad: 90}, classic.WarnColor},
		{"bad usage", bar.Params{Usage: 90, Limited: true, Limit: 95, Warn: 50, Bad: 90}, classic.BadColor},
		{"reached", bar.Params{Usage: 30, Limited: true, Limit: 80, Warn: 50, Bad: 90, Reached: true}, classic.BadColor},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.Fill(classic))
		})
	}
}
