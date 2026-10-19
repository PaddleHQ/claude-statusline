package style_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/featuretest/style"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "built-in styles",
			raw:  "5h: \033[33m██\033[0m\033[1m┃\033[0m\033[2m░░\033[0m 75%",
			want: "5h: ██┃░░ 75%\n    yyBdd",
		},
		{
			name: "every built-in letter",
			raw:  "\033[31ma\033[32mb\033[33mc\033[34md\033[35me\033[36mf\033[37mg\033[90mh\033[0m\033[1mi\033[0m\033[2mj",
			want: "abcdefghij\nrgybmcwkBd",
		},
		{
			name: "redundant resets and repeated codes do not matter",
			raw:  "\033[0m\033[0m\033[1m\033[1ma\033[0m\033[22mb\033[m",
			want: "ab\nB",
		},
		{
			name: "true color, palette color and a combination need a legend",
			raw:  "\033[38;2;148;226;173ma\033[0m \033[1m\033[38;5;33mb\033[0m \033[1;33mc\033[0m",
			want: "a b c\n1 2 3\n\n1 = #94e2ad\n2 = bold 256:33\n3 = bold yellow",
		},
		{
			name: "background and bright colors",
			raw:  "\033[44ma\033[0m\033[91mb\033[0m\033[48;5;208mc\033[49m",
			want: "abc\n123\n\n1 = bg:blue\n2 = bright-red\n3 = bg:256:208",
		},
		{
			name: "a second legend entry reuses its character",
			raw:  "\033[38;5;33ma\033[0mb\033[38;5;33mc",
			want: "abc\n1 1\n\n1 = 256:33",
		},
		{
			name: "wide runes take a continuation character when styled",
			raw:  "a\033[31m表\033[0m表z",
			want: "a表表z\n r_",
		},
		{
			name: "rows alternate text and style lines",
			raw:  "one\033[32mtwo\033[0m\nplain\n\033[2mthree",
			want: "onetwo\n   ggg\nplain\n\nthree\nddddd",
		},
		{
			name: "trailing unstyled spaces are dropped but styled ones stay",
			raw:  "a   \nb\033[41m  \033[0m  ",
			want: "a\n\nb  \n 11\n\n1 = bg:red",
		},
		{
			name: "zero-width characters stay with the character before",
			raw:  "a\u200bb",
			want: "a\u200bb\n",
		},
		{
			name: "no output has no rows",
			raw:  "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			block, err := style.Decode(tt.raw)
			require.NoError(t, err)
			got, err := block.Format()
			require.NoError(t, err)
			assert.Equal(t, strings.TrimRight(tt.want, "\n"), strings.TrimRight(got, "\n"))
		})
	}
}

func TestDecodeRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"a cursor movement", "a\033[2Ab", "ESC[2A"},
		{"a non CSI escape", "a\033]0;title\007", "does not start a CSI sequence"},
		{"an unterminated sequence", "a\033[3", "unterminated"},
		{"a lone escape at the end", "a\033", "does not start a CSI sequence"},
		{"an unknown SGR parameter", "\033[3ma", "SGR parameter 3"},
		{"a truncated true color", "\033[38;2;1;2ma", "SGR parameter 38"},
		{"a true color channel out of range", "\033[38;2;1;2;300ma", "SGR parameter 38"},
		{"a palette color out of range", "\033[38;5;256ma", "SGR parameter 38"},
		{"an unknown extended mode", "\033[38;9ma", "SGR parameter 38"},
		{"an extended color with no mode", "\033[38ma", "SGR parameter 38"},
		{"a non numeric parameter", "\033[1:2ma", "SGR parameter \"1:2\""},
		{"a zero-width character first", "\u200ba", "zero-width"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := style.Decode(tt.raw)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestParseMatchesDecode(t *testing.T) {
	t.Parallel()

	raw := "\033[1m\033[38;5;33m~/code\033[0m \033[2m·\033[0m \033[38;2;148;226;173m▰▰\033[0m\033[2m▱▱\033[0m 表\033[32m表\033[0m"
	decoded, err := style.Decode(raw)
	require.NoError(t, err)

	doc := "~/code · ▰▰▱▱ 表表\n111111 d 22dd   g_\n\n1 = bold 256:33\n2 = #94e2ad"
	parsed, err := style.Parse(doc)
	require.NoError(t, err)
	assert.True(t, decoded.Equal(parsed))
	assert.Equal(t, "~/code · ▰▰▱▱ 表表", decoded.Text())
	assert.True(t, decoded.Styled())
}

func TestParseTolerates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		doc  string
		want string
	}{
		{"a missing last style line", "a", "a"},
		{"trailing spaces on a text line", "a   \n", "a"},
		{"a styled trailing space written without its text", "a\n g\n\n", "a "},
		{"an empty document", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			block, err := style.Parse(tt.doc)
			require.NoError(t, err)
			assert.Equal(t, tt.want, block.Text())
		})
	}
}

func TestParseRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		doc  string
		want string
	}{
		{"an unknown style character", "ab\n 9", "unknown style: '9' in column 2"},
		{"redefining a built-in", "a\nr\n\nr = bold", "built in and cannot be redefined"},
		{"redefining the continuation character", "a\n_\n\n_ = bold", "built in and cannot be redefined"},
		{"defining a character twice", "a\n1\n\n1 = bold\n1 = dim", "defined twice"},
		{"an unused legend entry", "a\nd\n\n1 = bold", "not used by any row"},
		{"a malformed legend line", "a\nd\n\nbold", "not in the form"},
		{"an unknown legend style", "a\n1\n\n1 = sparkly", "unknown style \"sparkly\""},
		{"a bad hex color", "a\n1\n\n1 = #12", "unknown style \"#12\""},
		{"two foreground colors", "a\n1\n\n1 = red blue", "two colors of one kind"},
		{"an empty legend style", "a\n1\n\n1 = bg:", "unknown style"},
		{"a wide rune without its continuation", "表\nr", "wide character"},
		{"a wide rune with a continuation it must not have", "表\n _", "wide character"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := style.Parse(tt.doc)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestFormatErrors(t *testing.T) {
	t.Parallel()

	block, err := style.Decode("\033[31mred\033[0m\n\nnext")
	require.NoError(t, err)
	_, err = block.Format()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")

	var raw strings.Builder
	for i := range 60 {
		fmt.Fprintf(&raw, "\033[38;5;%dmx", i)
	}
	block, err = style.Decode(raw.String())
	require.NoError(t, err)
	_, err = block.Format()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "more styles")
}

func TestFormatLegendOrder(t *testing.T) {
	t.Parallel()

	block, err := style.Decode("\033[38;5;1ma\033[38;5;2mb\033[38;5;3mc\033[38;5;4md\033[38;5;5me\033[38;5;6mf\033[38;5;7mg\033[38;5;8mh\033[38;5;9mi\033[38;5;10mj\033[38;5;11mk")
	require.NoError(t, err)
	got, err := block.Format()
	require.NoError(t, err)
	assert.Contains(t, got, "abcdefghijk\n123456789ae")
	assert.Contains(t, got, "9 = 256:9\na = 256:10\ne = 256:11")
}

func TestEqual(t *testing.T) {
	t.Parallel()

	a, err := style.Decode("\033[31mred\033[0m")
	require.NoError(t, err)
	same, err := style.Decode("\033[0m\033[31mred")
	require.NoError(t, err)
	otherStyle, err := style.Decode("\033[32mred")
	require.NoError(t, err)
	otherText, err := style.Decode("\033[31mrex")
	require.NoError(t, err)
	shorter, err := style.Decode("\033[31mre")
	require.NoError(t, err)
	moreRows, err := style.Decode("\033[31mred\nred")
	require.NoError(t, err)

	assert.True(t, a.Equal(same))
	assert.False(t, a.Equal(otherStyle))
	assert.False(t, a.Equal(otherText))
	assert.False(t, a.Equal(shorter))
	assert.False(t, a.Equal(moreRows))
	assert.False(t, a.Equal(nil))
}
