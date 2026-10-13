package segment

import (
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/layout"
)

// families are the model families recognized by a substring of the model id, checked in order.
var families = []struct{ key, label string }{
	{"haiku", "Haiku"},
	{"sonnet", "Sonnet"},
	{"opus", "Opus"},
	{"fable", "Fable"},
	{"mythos", "Mythos"},
}

type model struct{}

func (m *model) Name() string { return nameModel }

func (m *model) Group() layout.Group { return layout.Identity }

func (m *model) Keys() []string { return []string{"model.effort"} }

func (m *model) Render(ctx *Context) string {
	name := ctx.Payload.Model.DisplayName
	level := ctx.Payload.Effort.Level
	// A skill or subagent can run a different model than the session's, which the payload does not reflect.
	if actual, ok := family(ctx.Transcript.ActualModel); ok {
		if session, _ := family(ctx.Payload.Model.ID); actual.key != session.key {
			name, level = actual.label+" (skill)", ""
		}
	}
	if name == "" {
		return ""
	}
	if level != "" && ctx.Config.Bool("model.effort") {
		name += " [" + level + "]"
	}
	return ctx.Theme.Paint(ctx.Theme.Text, name)
}

func family(id string) (f struct{ key, label string }, ok bool) {
	for _, f := range families {
		if strings.Contains(id, f.key) {
			return f, true
		}
	}
	return f, false
}
