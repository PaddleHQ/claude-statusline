package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSVGStyles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain text keeps the default foreground", "hi", `<tspan x="16">hi</tspan>`},
		{"bold", "\033[1mhi", `<tspan x="16" font-weight="bold">hi</tspan>`},
		{"dim", "\033[2mhi", `<tspan x="16" opacity="0.55">hi</tspan>`},
		{"italic", "\033[3mhi", `<tspan x="16" font-style="italic">hi</tspan>`},
		{"underline", "\033[4mhi", `<tspan x="16" text-decoration="underline">hi</tspan>`},
		{"basic foreground", "\033[32mhi", `<tspan x="16" fill="#3fb950">hi</tspan>`},
		{"bright foreground", "\033[91mhi", `<tspan x="16" fill="#ffa198">hi</tspan>`},
		{"256 color from the basic range", "\033[38;5;1mhi", `<tspan x="16" fill="#ff7b72">hi</tspan>`},
		{"256 color from the cube", "\033[38;5;33mhi", `<tspan x="16" fill="#0087ff">hi</tspan>`},
		{"256 color from the grayscale ramp", "\033[38;5;232mhi", `<tspan x="16" fill="#080808">hi</tspan>`},
		{"true color", "\033[38;2;255;128;0mhi", `<tspan x="16" fill="#ff8000">hi</tspan>`},
		{"stacked attributes", "\033[1m\033[38;5;33mhi", `<tspan x="16" fill="#0087ff" font-weight="bold">hi</tspan>`},
		{"combined parameters", "\033[1;4;31mhi", `<tspan x="16" fill="#ff7b72" font-weight="bold" text-decoration="underline">hi</tspan>`},
		{"reset clears everything", "\033[1m\033[31ma\033[0mb", `<tspan x="16" fill="#ff7b72" font-weight="bold">a</tspan><tspan x="25">b</tspan>`},
		{"empty parameters reset", "\033[1ma\033[mb", `<tspan x="16" font-weight="bold">a</tspan><tspan x="25">b</tspan>`},
		{"a later color replaces an earlier one", "\033[31m\033[34mhi", `<tspan x="16" fill="#58a6ff">hi</tspan>`},
		{"unknown codes are ignored", "\033[9mhi", `<tspan x="16">hi</tspan>`},
		{"other sequences are dropped", "\033[2Jhi", `<tspan x="16">hi</tspan>`},
		{"a truncated extended color is ignored", "\033[38;5mhi", `<tspan x="16">hi</tspan>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, string(Parse(tt.in).SVG(Dark)), tt.want)
		})
	}
}

func TestSVGLightThemeUsesItsOwnPalette(t *testing.T) {
	t.Parallel()

	svg := string(Parse("\033[32mhi").SVG(Light))
	assert.Contains(t, svg, `fill="#ffffff"`)
	assert.Contains(t, svg, `fill="#24292f"`)
	assert.Contains(t, svg, `<tspan x="16" fill="#1a7f37">hi</tspan>`)
}

func TestSVGEscapesXML(t *testing.T) {
	t.Parallel()

	svg := string(Parse(`a<b>&"c"`).SVG(Dark))
	assert.Contains(t, svg, "a&lt;b&gt;&amp;&quot;c&quot;")
	assert.NotContains(t, svg, "<b>")
}

func TestSVGLayout(t *testing.T) {
	t.Parallel()

	svg := string(Parse("abc\nde").SVG(Dark))
	assert.Contains(t, svg, `width="59" height="72"`, "the widest row sets the width, each row adds a line")
	assert.Contains(t, svg, `<text y="31">`)
	assert.Contains(t, svg, `<text y="51">`)
	assert.Contains(t, svg, `xml:space="preserve"`)
}

func TestSVGSkipsBlankRuns(t *testing.T) {
	t.Parallel()

	svg := string(Parse("a  \033[31mb").SVG(Dark))
	assert.Contains(t, svg, `<tspan x="16">a  </tspan><tspan x="43" fill="#ff7b72">b</tspan>`)

	svg = string(Parse("\033[31m   \033[0mb").SVG(Dark))
	assert.NotContains(t, svg, "   </tspan>")
}

func TestSVGDropsControlCharacters(t *testing.T) {
	t.Parallel()

	svg := string(Parse("a\x00\x07\tb").SVG(Dark))
	assert.Contains(t, svg, "<tspan x=\"16\">ab</tspan>")
}

func TestSVGIsDeterministic(t *testing.T) {
	t.Parallel()

	in := "\033[1m\033[38;5;33mpath\033[0m \033[2m·\033[0m x"
	first := Parse(in).SVG(Dark)
	second := Parse(in).SVG(Dark)
	assert.Equal(t, first, second)
}
