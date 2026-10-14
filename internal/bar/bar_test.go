package bar_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/bar"
	"github.com/PaddleHQ/claude-statusline/internal/theme"

	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, tt.want, tt.p.Render(tt.th))
		})
	}
}

func TestRenderWidth(t *testing.T) {
	t.Parallel()

	p := &bar.Params{Usage: 33, Target: 66, Width: 20, Warn: 20, Bad: 40, HardCap: 95}
	got := p.Render(markers.WithoutColor())
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
		{"below warn is good", 49, "<g>FF</><d>EE</>"},
		{"at warn is amber", 50, "<w>FF</><d>EE</>"},
		{"just below bad is amber", 89, "<w>FFF4</>"},
		{"at bad is red", 90, "<b>FFF5</>"},
		{"empty has no marker", 0, "<d>EEEE</>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &bar.Params{Usage: tt.usage, Target: 10, Width: 4, Warn: 50, Bad: 90, HardCap: 1, NoTarget: true}
			got := p.Render(markers)
			assert.Equal(t, tt.want, got)
			assert.NotContains(t, got, "M")
		})
	}
}

func TestRenderGradient(t *testing.T) {
	t.Parallel()

	th := theme.Theme{
		Good: "<g>", Warn: "<w>", Bad: "<b>", Reset: "</>",
		Full: "F", Empty: "E", MarkerGlyph: "M",
		Gradient: true, Ramp: [][3]int{{0, 0, 0}, {100, 100, 100}, {200, 200, 200}},
		EmptyColor: "<e>", MarkerColor: "<m>",
	}
	c := func(n int) string {
		return "\033[38;2;" + strings.Repeat(strconv.Itoa(n)+";", 2) + strconv.Itoa(n) + "m"
	}
	tests := []struct {
		name string
		p    bar.Params
		th   theme.Theme
		want string
	}{
		{"whole cells rounded to nearest", bar.Params{Usage: 40, Width: 4, Warn: 50, Bad: 90, NoTarget: true}, th, c(0) + "F</>" + c(66) + "F</><e>EE</>"},
		{"half a cell rounds up", bar.Params{Usage: 38, Width: 4, Warn: 50, Bad: 90, NoTarget: true}, th, c(0) + "F</>" + c(66) + "F</><e>EE</>"},
		{"below half a cell rounds down", bar.Params{Usage: 35, Width: 4, Warn: 50, Bad: 90, NoTarget: true}, th, c(0) + "F</><e>EEE</>"},
		{"marker replaces its cell", bar.Params{Usage: 100, Target: 50, Width: 3, Warn: 20, Bad: 40, HardCap: 95}, th, c(0) + "F</><m>M</>" + c(200) + "F</>"},
		{"no ramp color without color", bar.Params{Usage: 50, Target: 0, Width: 2, Warn: 20, Bad: 40, HardCap: 95}, th.WithoutColor(), "ME"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.Render(tt.th))
		})
	}
}

func TestPercentColor(t *testing.T) {
	t.Parallel()

	p := &bar.Params{Usage: 71, Target: 50, Width: 2, Warn: 20, Bad: 40, HardCap: 95}
	assert.Empty(t, p.PercentColor(markers), "solid bars color the fill, not the text")
	grad := markers
	grad.Gradient = true
	assert.Equal(t, "<w>", p.PercentColor(grad))
	fill := &bar.Params{Usage: 10, Width: 2, Warn: 50, Bad: 90, NoTarget: true}
	assert.Equal(t, "<g>", fill.Fill(markers))
}

func TestRenderMergesRuns(t *testing.T) {
	t.Parallel()

	grad := theme.Theme{
		Reset: "</>", Full: "F", Empty: "E", MarkerGlyph: "M",
		Gradient: true, Ramp: [][3]int{{0, 0, 0}, {100, 100, 100}, {200, 200, 200}},
		EmptyColor: "<e>", MarkerColor: "<m>",
	}
	ramp := grad.RampColor
	tests := []struct {
		name string
		p    bar.Params
		th   theme.Theme
		want string
	}{
		{"solid with a marker in the middle", bar.Params{Usage: 40, Target: 50, Width: 6, Warn: 20, Bad: 40, HardCap: 95}, markers, "<g>FF3</><m>M</><d>EE</>"},
		{"all empty", bar.Params{Usage: 0, Target: 0, Width: 4, Warn: 50, Bad: 90, NoTarget: true}, markers, "<d>EEEE</>"},
		{"all full", bar.Params{Usage: 100, Width: 4, Warn: 150, Bad: 200, NoTarget: true}, markers, "<g>FFFF</>"},
		{
			"gradient keeps ramp cells apart and merges the empty tail",
			bar.Params{Usage: 50, Width: 4, Warn: 50, Bad: 90, NoTarget: true},
			grad,
			ramp(0, 4) + "F</>" + ramp(1, 4) + "F</><e>EE</>",
		},
		{"no color has no escapes", bar.Params{Usage: 50, Target: 75, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, markers.WithoutColor(), "FFEM"},
		{"no color gradient has no escapes", bar.Params{Usage: 50, Width: 4, Warn: 50, Bad: 90, NoTarget: true}, grad.WithoutColor(), "FFEE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.Render(tt.th))
		})
	}
}

func TestFillAgainstThePaceAtTheDefaultThresholds(t *testing.T) {
	t.Parallel()

	classic := theme.Load("classic", "", "")
	tests := []struct {
		name          string
		usage, target int
		want          string
	}{
		{"60% with half the window elapsed is green", 60, 50, classic.Good},
		{"exactly the warn deviation is still green", 70, 50, classic.Good},
		{"75% with half the window elapsed is amber", 75, 50, classic.Warn},
		{"exactly the bad deviation is still amber", 90, 50, classic.Warn},
		{"92% with half the window elapsed is red", 92, 50, classic.Bad},
		{"5% with half the window elapsed, far behind the pace, is red", 5, 50, classic.Bad},
		{"96% is red from the hard cap even on the pace", 96, 96, classic.Bad},
		{"96% a quarter of the window ahead of the pace is red", 96, 5, classic.Bad},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &bar.Params{Usage: tt.usage, Target: tt.target, Width: 10, Warn: 20, Bad: 40, HardCap: 95}
			assert.Equal(t, tt.want, p.Fill(classic))
		})
	}
}

func TestFillByUsageAtTheDefaultContextThresholds(t *testing.T) {
	t.Parallel()

	classic := theme.Load("classic", "", "")
	tests := []struct {
		usage int
		want  string
	}{
		{30, classic.Good},
		{49, classic.Good},
		{50, classic.Warn},
		{89, classic.Warn},
		{90, classic.Bad},
		{125, classic.Bad},
	}
	for _, tt := range tests {
		p := &bar.Params{Usage: tt.usage, Width: 10, Warn: 50, Bad: 90, NoTarget: true}
		assert.Equal(t, tt.want, p.Fill(classic), "usage %d", tt.usage)
	}
}
