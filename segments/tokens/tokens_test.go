package tokens_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/segments/tokens"
	"github.com/PaddleHQ/claude-statusline/segments/tokens/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFormatTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0k"},
		{1049, "1.0k"},
		{1500, "1.5k"},
		{999999, "999.9k"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{12345678, "12.3M"},
		{64506, "64.5k"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tokens.FormatTokens(tt.n))
		})
	}
}

func build(t *testing.T, in, out *int64) *tokens.Totals {
	t.Helper()

	sess := mocks.NewSession(t)
	sess.EXPECT().TotalInput().RunAndReturn(func() (int64, bool) { return value(in) }).Maybe()
	sess.EXPECT().TotalOutput().RunAndReturn(func() (int64, bool) { return value(out) }).Maybe()
	paint := mocks.NewPainter(t)
	paint.EXPECT().Text(mock.Anything).RunAndReturn(func(s string) string { return "<t>" + s + "</>" }).Maybe()
	return tokens.New(tokens.Deps{Session: sess, Paint: paint, Separator: " <m>·</> "})
}

func value(p *int64) (int64, bool) {
	if p == nil {
		return 0, false
	}
	return *p, true
}

func TestRender(t *testing.T) {
	t.Parallel()

	i := func(n int64) *int64 { return &n }
	tests := []struct {
		name    string
		in, out *int64
		want    string
	}{
		{"both", i(1500), i(250), "<t>↓ 1.5k</> <m>·</> <t>↑ 250</>"},
		{"input only", i(1500), nil, "<t>↓ 1.5k</>"},
		{"output only", nil, i(7), "<t>↑ 7</>"},
		{"zero is shown", i(0), nil, "<t>↓ 0</>"},
		{"output zero alone is shown", nil, i(0), "<t>↑ 0</>"},
		{"below a thousand as is, exactly a thousand as 1.0k", i(999), i(1000), "<t>↓ 999</> <m>·</> <t>↑ 1.0k</>"},
		{"thousands with a k suffix", i(1500), i(999999), "<t>↓ 1.5k</> <m>·</> <t>↑ 999.9k</>"},
		{"millions with an M suffix", i(1000000), i(1500000), "<t>↓ 1.0M</> <m>·</> <t>↑ 1.5M</>"},
		{"neither hides the segment", nil, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, build(t, tt.in, tt.out).Render())
		})
	}
}

func TestNameKeysAndHelp(t *testing.T) {
	t.Parallel()

	seg := build(t, nil, nil)
	assert.Equal(t, "tokens", seg.Name())
	assert.Empty(t, seg.Keys())
	assert.NotEmpty(t, seg.Help())
}
