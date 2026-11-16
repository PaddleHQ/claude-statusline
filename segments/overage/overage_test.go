package overage_test

import (
	"testing"
	"time"

	"github.com/PaddleHQ/claude-statusline/segments/overage"
	"github.com/PaddleHQ/claude-statusline/segments/overage/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var now = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)

type world struct {
	pct          *float64
	usedUSD      *float64
	limitUSD     *float64
	month        float64
	atLimit      bool
	nerd         bool
	limitsCalled *int
}

func (w world) build(t *testing.T) *overage.Overage {
	t.Helper()

	spend := mocks.NewSpend(t)
	if w.pct != nil {
		spend.EXPECT().SpendPercentage().Return(*w.pct, true).Maybe()
	} else {
		spend.EXPECT().SpendPercentage().Return(0, false).Maybe()
	}
	if w.usedUSD != nil && w.limitUSD != nil {
		spend.EXPECT().SpendUSD().Return(*w.usedUSD, *w.limitUSD, true).Maybe()
	} else {
		spend.EXPECT().SpendUSD().Return(0, 0, false).Maybe()
	}
	ledger := mocks.NewLedger(t)
	ledger.EXPECT().Month(now).Return("2026-11", w.month).Maybe()
	limits := mocks.NewLimits(t)
	limits.EXPECT().AtLimit().Return(w.atLimit).Maybe()
	icons := mocks.NewIcons(t)
	icons.EXPECT().Pick(mock.Anything, mock.Anything).RunAndReturn(func(plain, nerd string) string {
		if w.nerd {
			return nerd
		}
		return plain
	}).Maybe()
	paint := mocks.NewPainter(t)
	paint.EXPECT().Text(mock.Anything).RunAndReturn(func(s string) string { return "<t>" + s + "</>" }).Maybe()
	paint.EXPECT().Warn(mock.Anything).RunAndReturn(func(s string) string { return "<w>" + s + "</>" }).Maybe()
	paint.EXPECT().Dim(mock.Anything).RunAndReturn(func(s string) string { return "<d>" + s + "</>" }).Maybe()

	return overage.New(overage.Deps{
		Spend:  spend,
		Ledger: ledger,
		Limits: func() overage.Limits {
			if w.limitsCalled != nil {
				*w.limitsCalled++
			}
			return limits
		},
		Paint: paint,
		Icons: icons,
		Now:   now,
	})
}

func TestRenderEstimate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		month float64
		limit bool
		want  string
	}{
		{"zero is dimmed, never hidden", 0, false, "<d>» ~$0.00</>"},
		{"zero at the limit is still dimmed", 0, true, "<d>» ~$0.00</>"},
		{"spend with no limit at 100% is plain", 5, false, "<t>» ~$5.00</>"},
		{"spend while a limit is at 100% is amber", 5, true, "<w>» ~$5.00</>"},
		{"rounds to cents", 3.456, false, "<t>» ~$3.46</>"},
		{"rounds down", 3.454, false, "<t>» ~$3.45</>"},
		{"a fraction of a cent rounds to zero and is dimmed", 0.004, true, "<d>» ~$0.00</>"},
		{"a fraction that rounds up to a cent is shown", 0.005, false, "<t>» ~$0.01</>"},
		{"large", 1234.5, true, "<w>» ~$1234.50</>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, world{month: tt.month, atLimit: tt.limit}.build(t).Render())
		})
	}
}

func TestRenderEstimateOnlyReadsTheLimitsWhenThereIsSpend(t *testing.T) {
	t.Parallel()

	var calls int
	world{month: 0, atLimit: true, limitsCalled: &calls}.build(t).Render()
	assert.Zero(t, calls)

	world{month: 2, atLimit: true, limitsCalled: &calls}.build(t).Render()
	assert.Equal(t, 1, calls)
}

func TestRenderReportedSpendLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		w    world
		want string
	}{
		{"dollars used of the limit is amber", world{usedUSD: new(12.5), limitUSD: new(50.0)}, "<w>» $12.50/$50</>"},
		{"nothing used is dimmed", world{usedUSD: new(0.0), limitUSD: new(50.0)}, "<d>» $0.00/$50</>"},
		{"a limit with cents keeps them", world{usedUSD: new(1.0), limitUSD: new(12.5)}, "<w>» $1.00/$12.50</>"},
		{"used rounds to cents", world{usedUSD: new(1.234), limitUSD: new(20.0)}, "<w>» $1.23/$20</>"},
		{"a fraction of a cent counts as nothing used", world{usedUSD: new(0.004), limitUSD: new(20.0)}, "<d>» $0.00/$20</>"},
		{"dollars win over the percentage", world{usedUSD: new(5.0), limitUSD: new(50.0), pct: new(10.0)}, "<w>» $5.00/$50</>"},
		{"a percentage alone", world{pct: new(40.0)}, "<w>» 40%</>"},
		{"a percentage rounds", world{pct: new(39.6)}, "<w>» 40%</>"},
		{"zero percent is dimmed", world{pct: new(0.0)}, "<d>» 0%</>"},
		{"under half a percent is dimmed", world{pct: new(0.4)}, "<d>» 0%</>"},
		{"the reported spend ignores the month total and the limits", world{pct: new(40.0), month: 9, atLimit: true}, "<w>» 40%</>"},
		{"dollars without a limit figure fall back to the percentage", world{usedUSD: new(5.0), pct: new(10.0)}, "<w>» 10%</>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.w.build(t).Render())
		})
	}
}

func TestRenderNerdLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "<t>\uF071 ~$5.00</>", world{month: 5, nerd: true}.build(t).Render())
	assert.Equal(t, "<w>\uF071 40%</>", world{pct: new(40.0), nerd: true}.build(t).Render())
}

func TestNameAndKeys(t *testing.T) {
	t.Parallel()

	seg := world{}.build(t)
	assert.Equal(t, "overage", seg.Name())
	assert.Empty(t, seg.Keys())
}

func TestHelp(t *testing.T) {
	t.Parallel()

	help := world{}.build(t).Help()
	assert.Contains(t, help, "calendar month (UTC)")
	assert.Contains(t, help, "estimate, not a bill")
	assert.Contains(t, help, "spend limit")
}
