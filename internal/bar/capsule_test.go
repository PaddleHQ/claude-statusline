package bar_test

import (
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/bar"
	"github.com/PaddleHQ/claude-statusline/internal/theme"

	"github.com/stretchr/testify/assert"
)

var capsule = theme.Theme{
	Good: "<g>", WarnColor: "<w>", BadColor: "<b>", Reset: "</>",
	Full: "F", Empty: "E", MarkerGlyph: "X", Capsule: true,
	EmptyColor: "<e>", MarkerColor: "<m>", Muted: "<mu>", Marker: "<mk>",
}

// glyphs maps L/M/R to the filled first, middle and last capsule glyphs and l/m/r to the empty ones.
func glyphs(s string) string {
	return strings.NewReplacer(
		"L", "\uee03", "M", "\uee04", "R", "\uee05",
		"l", "\uee00", "m", "\uee01", "r", "\uee02",
	).Replace(s)
}

func paint(color, s string) string { return color + glyphs(s) + "</>" }

func TestRenderCapsule(t *testing.T) {
	t.Parallel()

	targeted := func(usage, target, width int) bar.Params {
		return bar.Params{Usage: usage, Target: target, Width: width, Warn: 20, Bad: 40, HardCap: 95}
	}
	untargeted := func(usage, width int) bar.Params {
		return bar.Params{Usage: usage, Width: width, Warn: 50, Bad: 90, NoTarget: true}
	}
	tests := []struct {
		name string
		p    bar.Params
		want string
	}{
		{"width 1 empty is a left cap only", untargeted(0, 1), paint("<e>", "l")},
		{"width 1 full is a left cap only", untargeted(100, 1), paint("<b>", "L")},
		{"width 2 empty", untargeted(0, 2), paint("<e>", "lr")},
		{"width 2 full", untargeted(100, 2), paint("<b>", "LR")},
		{"empty, no target", untargeted(0, 4), paint("<e>", "lmmr")},
		{"30% no target, good", untargeted(30, 10), paint("<g>", "LMM") + paint("<e>", "mmmmmmr")},
		{"on pace with the marker inside the fill", targeted(40, 30, 10), paint("<g>", "LMM") + paint("<m>", "M") + paint("<e>", "mmmmmr")},
		{"marker in the empty part", targeted(20, 50, 10), paint("<w>", "LM") + paint("<e>", "mmm") + paint("<m>", "M") + paint("<e>", "mmmr")},
		{"marker on the first cell", targeted(0, 0, 10), paint("<m>", "L") + paint("<e>", "mmmmmmmmr")},
		{"marker on the last cell", targeted(50, 100, 10), paint("<b>", "LMMMM") + paint("<e>", "mmmm") + paint("<m>", "R")},
		{"full", untargeted(100, 4), paint("<b>", "LMMR")},
		{"over 100", untargeted(130, 4), paint("<b>", "LMMR")},
		{"rounding up", untargeted(35, 10), paint("<g>", "LMMM") + paint("<e>", "mmmmmr")},
		{"rounding down", untargeted(34, 10), paint("<g>", "LMM") + paint("<e>", "mmmmmmr")},
		{"hard cap", targeted(96, 95, 10), paint("<b>", "LMMMMMMMM") + paint("<m>", "R")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.p.Nerd = true
			assert.Equal(t, tt.want, tt.p.Render(capsule))
		})
	}
}

func TestRenderCapsuleFallsBackToSolidWithoutNerd(t *testing.T) {
	t.Parallel()

	solid := capsule
	solid.Capsule = false
	p := bar.Params{Usage: 40, Target: 30, Width: 10, Warn: 20, Bad: 40, HardCap: 95}
	assert.Equal(t, p.Render(solid), p.Render(capsule))

	capsuleMeter := &bar.Meter{Theme: capsule, Width: 10}
	solidMeter := &bar.Meter{Theme: solid, Width: 10}
	assert.Equal(t, solidMeter.Usage(60, 50, 90), capsuleMeter.Usage(60, 50, 90), "the percentage is not colored either")
	assert.Equal(t, solidMeter.Pace(40, 30, 20, 40, 95), capsuleMeter.Pace(40, 30, 20, 40, 95))
}

func TestMeterCopiesNerd(t *testing.T) {
	t.Parallel()

	m := &bar.Meter{Theme: capsule, Width: 4, Nerd: true}
	assert.Contains(t, m.Usage(100, 50, 90), "\uee05")
	m.Nerd = false
	assert.NotContains(t, m.Usage(100, 50, 90), "\uee05")
}

func TestPercentColorOnCapsule(t *testing.T) {
	t.Parallel()

	p := bar.Params{Usage: 60, Width: 4, Warn: 50, Bad: 90, NoTarget: true, Nerd: true}
	assert.Equal(t, "<w>", p.PercentColor(capsule))

	p.Nerd = false
	assert.Empty(t, p.PercentColor(capsule), "without Nerd the bar is solid, so the percentage stays uncolored")
}
