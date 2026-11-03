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
			assert.Equal(t, tt.want, tt.p.Render(capsule))
		})
	}
}

func TestRenderCapsuleOfOneCellUsesTheThemeGlyphs(t *testing.T) {
	t.Parallel()

	solid := capsule
	solid.Capsule = false
	for _, usage := range []int{0, 100} {
		p := bar.Params{Usage: usage, Width: 1, Warn: 50, Bad: 90, NoTarget: true}
		assert.Equal(t, p.Render(solid), p.Render(capsule), "usage %d", usage)
		assert.Empty(t, p.PercentColor(capsule), "usage %d", usage)
	}
}

func TestMeterDrawsCapsules(t *testing.T) {
	t.Parallel()

	m := &bar.Meter{Theme: capsule, Width: 4}
	assert.Equal(t, paint("<w>", "LM")+paint("<e>", "mr")+" <w>60%</>", m.Usage(60, 50, 90))
}

func TestPercentColorOnCapsule(t *testing.T) {
	t.Parallel()

	p := bar.Params{Usage: 60, Width: 4, Warn: 50, Bad: 90, NoTarget: true}
	assert.Equal(t, "<w>", p.PercentColor(capsule))
}
