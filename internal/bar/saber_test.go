package bar_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/bar"
	"github.com/PaddleHQ/claude-statusline/internal/theme"

	"github.com/stretchr/testify/assert"
)

var blade = theme.Theme{
	Good: "<g>", WarnColor: "<w>", BadColor: "<b>", Reset: "</>",
	Full: "F", Empty: "E", MarkerGlyph: "M", Tip: "T",
	Saber: true, Blade: [][3]int{{0, 0, 0}, {100, 100, 100}, {200, 200, 200}}, Glow: 50, GlowCells: 2,
	EmptyColor: "<e>", MarkerColor: "<m>",
}

func TestRenderSaber(t *testing.T) {
	t.Parallel()

	c := blade.BladeColor
	tests := []struct {
		name string
		p    bar.Params
		th   theme.Theme
		want string
	}{
		{"whole cells", bar.Params{Usage: 50, Width: 4, Warn: 60, Bad: 90, NoTarget: true}, blade, c(0, 0, 2) + "F</>" + c(0, 1, 2) + "F</><e>EE</>"},
		{"an odd half count ends in the tip", bar.Params{Usage: 37, Width: 4, Warn: 60, Bad: 90, NoTarget: true}, blade, c(0, 0, 2) + "F</>" + c(0, 1, 2) + "T</><e>EE</>"},
		{"a lone tip", bar.Params{Usage: 13, Width: 4, Warn: 60, Bad: 90, NoTarget: true}, blade, c(0, 0, 1) + "T</><e>EEE</>"},
		{"below a quarter cell is empty", bar.Params{Usage: 3, Width: 4, Warn: 60, Bad: 90, NoTarget: true}, blade, "<e>EEEE</>"},
		{"a long blade keeps the plain color until the glow window", bar.Params{Usage: 63, Width: 4, Warn: 80, Bad: 90, NoTarget: true}, blade, c(0, 0, 3) + "F</>" + c(0, 1, 3) + "F</>" + c(0, 2, 3) + "T</><e>E</>"},
		{"full bar", bar.Params{Usage: 100, Width: 2, Warn: 150, Bad: 200, NoTarget: true}, blade, c(0, 0, 2) + "F</>" + c(0, 1, 2) + "F</>"},
		{"over a hundred is capped", bar.Params{Usage: 130, Width: 2, Warn: 150, Bad: 200, NoTarget: true}, blade, c(0, 0, 2) + "F</>" + c(0, 1, 2) + "F</>"},
		{"marker replaces its cell and the blade still counts it", bar.Params{Usage: 100, Target: 50, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, blade, c(2, 0, 4) + "FF</><m>M</>" + c(2, 3, 4) + "F</>"},
		{"marker over an empty cell", bar.Params{Usage: 0, Target: 50, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, blade, "<e>EE</><m>M</><e>E</>"},
		{"no color keeps the glyphs", bar.Params{Usage: 37, Target: 75, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, blade.WithoutColor(), "FTEM"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.Render(tt.th))
		})
	}
}

func TestRenderSaberPicksTheBladeByState(t *testing.T) {
	t.Parallel()

	plainBlade := blade
	plainBlade.Glow = 0
	tests := []struct {
		name  string
		p     bar.Params
		state int
	}{
		{"good without a target", bar.Params{Usage: 25, Width: 4, Warn: 50, Bad: 90, NoTarget: true}, 0},
		{"warn without a target", bar.Params{Usage: 50, Width: 4, Warn: 50, Bad: 90, NoTarget: true}, 1},
		{"bad without a target", bar.Params{Usage: 90, Width: 4, Warn: 50, Bad: 90, NoTarget: true}, 2},
		{"good on the pace", bar.Params{Usage: 25, Target: 25, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, 0},
		{"warn ahead of the pace", bar.Params{Usage: 50, Target: 25, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, 1},
		{"warn behind the pace", bar.Params{Usage: 25, Target: 50, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, 1},
		{"bad far from the pace", bar.Params{Usage: 75, Target: 25, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, 2},
		{"bad at the hard cap", bar.Params{Usage: 95, Target: 90, Width: 4, Warn: 20, Bad: 40, HardCap: 95}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.p.Render(plainBlade)
			assert.Contains(t, got, plainBlade.BladeColor(tt.state, 0, 1)+"F", "the first lit cell is the state's blade")
			assert.Equal(t, []string{"<g>", "<w>", "<b>"}[tt.state], tt.p.Fill(blade))
		})
	}
}

func TestPercentColorForASaberTheme(t *testing.T) {
	t.Parallel()

	p := &bar.Params{Usage: 71, Target: 50, Width: 2, Warn: 20, Bad: 40, HardCap: 95}
	assert.Equal(t, "<w>", p.PercentColor(blade))
	assert.Empty(t, p.PercentColor(blade.WithoutColor()), "NO_COLOR leaves the text plain")
}

func TestMeterColorsThePercentageByStateOnASaberTheme(t *testing.T) {
	t.Parallel()

	saber := theme.Load("saber", "", "")
	params := &bar.Params{Usage: 95, Width: 4, Warn: 70, Bad: 90, NoTarget: true}

	m := &bar.Meter{Theme: saber, Width: 4}
	assert.Equal(t, params.Render(saber)+" "+saber.BadColor+"95%"+saber.Reset, m.Usage(95, 70, 90))
}
