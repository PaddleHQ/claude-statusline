// Package overage provides the overage segment: extra-usage spend this calendar month.
package overage

import (
	"fmt"
	"math"
	"time"
)

const (
	name           = "overage"
	centsPerDollar = 100
)

//go:generate go run github.com/vektra/mockery/v2 --name=Spend

// Spend is the spend limit that Claude Code reports for some logins.
type Spend interface {
	// SpendPercentage is how much of the spend limit is used; false means no spend limit was reported.
	SpendPercentage() (float64, bool)
	// SpendUSD is the amount used and the limit in US dollars; false means the limit is not in dollars.
	SpendUSD() (used, limit float64, ok bool)
}

//go:generate go run github.com/vektra/mockery/v2 --name=Ledger

// Ledger is the estimate of this month's extra-usage spend, shared between sessions.
type Ledger interface {
	// Month is the UTC calendar month containing now, and the estimated spend in it in US dollars.
	Month(now time.Time) (key string, usd float64)
}

//go:generate go run github.com/vektra/mockery/v2 --name=Limits

// Limits is the state of the account's usage windows.
type Limits interface {
	// AtLimit reports whether any window has reached 100%.
	AtLimit() bool
}

//go:generate go run github.com/vektra/mockery/v2 --name=Painter

// Painter colors text.
type Painter interface {
	Text(s string) string
	Warn(s string) string
	Dim(s string) string
}

//go:generate go run github.com/vektra/mockery/v2 --name=Icons

// Icons picks a label for the user's icon set.
type Icons interface {
	Pick(plain, nerd string) string
}

// Deps is everything the segment is built from. Limits is a function so that the readings are only filtered when the
// segment needs them.
type Deps struct {
	Spend  Spend
	Ledger Ledger
	Limits func() Limits
	Paint  Painter
	Icons  Icons
	Now    time.Time
}

// Overage shows extra-usage spend.
type Overage struct{ d Deps }

// New builds the overage segment.
func New(d Deps) *Overage { return &Overage{d: d} }

// Name is the segment's name in the segments list.
func (o *Overage) Name() string { return name }

// Help says what the segment shows, for the help command.
func (o *Overage) Help() string {
	return "Extra-usage spend this calendar month (UTC), inferred from session cost at API prices while a 5h or 7d limit " +
		"is at 100%: an estimate, not a bill. Amber while a limit is at 100% and the figure is above zero, dim at zero. " +
		"Logins that report a spend limit show that instead."
}

// Keys lists the config keys the segment reads; it has none.
func (o *Overage) Keys() []string { return nil }

// Render returns the segment's text. It is never hidden: with nothing spent it is dimmed.
func (o *Overage) Render() string {
	label := o.d.Icons.Pick("\u00BB ", "\uF071 ")
	if text, spent, ok := o.reported(); ok {
		if spent {
			return o.d.Paint.Warn(label + text)
		}
		return o.d.Paint.Dim(label + text)
	}
	_, usd := o.d.Ledger.Month(o.d.Now)
	usd = cents(usd)
	text := label + fmt.Sprintf("~$%.2f", usd)
	switch {
	case usd == 0:
		return o.d.Paint.Dim(text)
	case o.d.Limits().AtLimit():
		return o.d.Paint.Warn(text)
	default:
		return o.d.Paint.Text(text)
	}
}

// reported is the text of the spend limit the payload reports, and whether anything has been spent against it.
func (o *Overage) reported() (text string, spent, ok bool) {
	if used, limit, ok := o.d.Spend.SpendUSD(); ok {
		used = cents(used)
		return fmt.Sprintf("$%.2f/%s", used, dollars(limit)), used > 0, true
	}
	if pct, ok := o.d.Spend.SpendPercentage(); ok {
		pct = math.Round(pct)
		return fmt.Sprintf("%.0f%%", pct), pct > 0, true
	}
	return "", false, false
}

// cents rounds to whole cents, so the color agrees with the figure shown.
func cents(usd float64) float64 { return math.Round(usd*centsPerDollar) / centsPerDollar }

// dollars formats a limit without cents when it has none: $50, but $12.50.
func dollars(usd float64) string {
	usd = cents(usd)
	if usd == math.Trunc(usd) {
		return fmt.Sprintf("$%.0f", usd)
	}
	return fmt.Sprintf("$%.2f", usd)
}
