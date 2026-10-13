package segment

import (
	"strconv"

	"github.com/PaddleHQ/claude-statusline/internal/bar"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
)

const (
	fiveHourSeconds = 5 * 60 * 60
	sevenDaySeconds = 7 * 24 * 60 * 60
)

// rate draws one usage window as "<label>: <bar> <NN>%".
type rate struct {
	name    string
	seconds int64
	reading func(*Context) *ratecache.Reading
}

func (r rate) Name() string { return r.name }

func (r rate) Group() layout.Group { return layout.Rates }

func (r rate) Keys() []string {
	return []string{r.name + ".label", r.name + ".width", r.name + ".warn", r.name + ".bad", r.name + ".hard_cap"}
}

func (r rate) Render(ctx *Context) string {
	reading := r.reading(ctx)
	if reading == nil {
		return ""
	}
	cfg := ctx.Config
	b := bar.Render(bar.Params{
		Usage:   int(reading.Used),
		Target:  bar.ElapsedPercent(ctx.Now.Unix(), reading.ResetsAt, r.seconds),
		Width:   bar.ResponsiveWidth(cfg.Int(r.name+".width"), ctx.Columns),
		Warn:    cfg.Int(r.name + ".warn"),
		Bad:     cfg.Int(r.name + ".bad"),
		HardCap: cfg.Int(r.name + ".hard_cap"),
	}, ctx.Theme)
	label := ctx.Theme.Paint(ctx.Theme.Text, cfg.String(r.name+".label")+": ")
	return label + b + " " + strconv.FormatInt(reading.Used, 10) + "%"
}
