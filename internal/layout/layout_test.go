package layout_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/PaddleHQ/claude-statusline/internal/layout"
)

const (
	sep    = " · "
	anchor = "↳"
)

func render(columns int, items ...layout.Item) string {
	return layout.Render(items, layout.Options{Columns: columns, Separator: sep, Anchor: anchor})
}

func id(s string) layout.Item    { return layout.Item{Group: layout.Identity, Text: s} }
func usage(s string) layout.Item { return layout.Item{Group: layout.Usage, Text: s} }
func rates(s string) layout.Item { return layout.Item{Group: layout.Rates, Text: s} }
func extra(s string) layout.Item { return layout.Item{Group: layout.Extra, Text: s} }

func TestWidth(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 5, layout.Width("hello"))
	assert.Equal(t, 5, layout.Width("\x1b[1m\x1b[38;5;33mhello\x1b[0m"))
	assert.Equal(t, 3, layout.Width("↓ 1"), "multi-byte runes count once")
	assert.Equal(t, 0, layout.Width(""))
}

func TestRenderOneRow(t *testing.T) {
	t.Parallel()

	items := []layout.Item{id("/srv/app"), id("Opus"), usage("↓ 1k"), rates("5h: 45%"), extra("hook")}
	const joined = "/srv/app · Opus · ↓ 1k · 5h: 45% · hook"

	assert.Equal(t, joined, render(len(joined), items...), "exactly the width still fits")
	assert.Equal(t, joined, render(999, items...))
	assert.Equal(t, joined, render(0, items...), "an unknown width never wraps")
	assert.Empty(t, render(40))
}

func TestRenderWrapped(t *testing.T) {
	t.Parallel()

	path, model, ctx := "/srv/app", "Opus 4.7 [high]", "ctx: 42%"
	tok, rate := "↓ 1.5k · ↑ 23.4k · ⧉ 85%", "5h: 45% · 7d: 40%"
	// path + separator is 11 wide, so continuation rows indent by 11.
	pad := "          "

	tests := []struct {
		name  string
		cols  int
		items []layout.Item
		want  string
	}{
		{
			"usage row with the rates aligned under the model",
			60,
			[]layout.Item{id(path), id(model), id(ctx), usage(tok), rates(rate)},
			path + sep + model + sep + ctx + "\n" + tok + " " + rate,
		},
		{
			"rates on their own indented row when they do not fit beside the tokens",
			40,
			[]layout.Item{id(path), id(model), usage(tok), rates(rate)},
			path + sep + model + "\n" + tok + "\n" + anchor + pad + rate,
		},
		{
			"identity wraps with the indent",
			36,
			[]layout.Item{id(path), id(model), id(ctx)},
			path + sep + model + "\n" + anchor + pad + ctx,
		},
		{
			"rates alone are indented",
			40,
			[]layout.Item{id(path), id(model), id(ctx), rates(rate)},
			path + sep + model + sep + ctx + "\n" + anchor + pad + rate,
		},
		{
			"tokens alone sit flush left",
			40,
			[]layout.Item{id(path), id(model), id(ctx), usage(tok)},
			path + sep + model + sep + ctx + "\n" + tok,
		},
		{
			"extra items take their own row",
			40,
			[]layout.Item{id(path), id(model), id(ctx), extra("one"), extra("two")},
			path + sep + model + sep + ctx + "\none" + sep + "two",
		},
		{
			"extra items alone",
			20,
			[]layout.Item{extra("one"), extra("two")},
			"one" + sep + "two",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := render(tt.cols, tt.items...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRenderIndentDisabledWhenTooNarrow(t *testing.T) {
	t.Parallel()

	got := render(30, id("/srv/app"), id("Opus 4.7 [high]"), id("ctx: 42%"), usage("tok"), rates("5h: 45% · 7d: 40%"))
	want := "/srv/app · Opus 4.7 [high]\nctx: 42%\ntok 5h: 45% · 7d: 40%"
	assert.Equal(t, want, got)

	got = render(23, id("/srv/app"), id("Opus 4.7 [high]"), rates("5h: 45%"))
	assert.Equal(t, "/srv/app\nOpus 4.7 [high]\n5h: 45%", got, "no room left after the indent, so no anchor")
}

func TestRenderPacksIdentityRowsGreedily(t *testing.T) {
	t.Parallel()

	a, b, c := strings.Repeat("a", 12), strings.Repeat("b", 12), strings.Repeat("c", 12)
	got := render(40, id(a), id(b), id(c), id("dd"))
	assert.Equal(t, a+sep+b+"\n"+anchor+strings.Repeat(" ", 14)+c+sep+"dd", got)
}

func TestRenderMeasuresVisibleWidth(t *testing.T) {
	t.Parallel()

	red := "\x1b[31mred\x1b[0m"
	assert.Equal(t, red+sep+red, render(9, id(red), id(red)), "colour sequences take no room")
	assert.Equal(t, red+"\n"+red, render(8, id(red), id(red)))
}
