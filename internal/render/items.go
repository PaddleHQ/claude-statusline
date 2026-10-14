package render

import (
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/hook"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
)

// items renders the configured segments in order, then places the hooks among them.
//
// A hook named in the segments list renders there, and replaces a built-in of the same name.
// Every other hook goes to its slot: after the context segment, after the usage group, or at the end.
func (l *Line) items() []layout.Item {
	names := strings.Fields(l.Providers.Config.String(keySegments))
	byName, unplaced := partitionHooks(l.Hooks, names)

	var out []layout.Item
	ctxAt, modelAt := -1, -1
	for _, name := range names {
		builtin, isBuiltin := l.lookup(name)
		if hooks := byName[name]; len(hooks) > 0 {
			out = appendNamedHooks(out, hooks, builtin, isBuiltin)
			continue
		}
		if !isBuiltin {
			continue
		}
		text := builtin.Render(l.Session)
		if text == "" {
			continue
		}
		out = append(out, layout.Item{Group: builtin.Group(), Text: text})
		switch name {
		case nameCtx:
			ctxAt = len(out) - 1
		case nameModel:
			modelAt = len(out) - 1
		}
	}
	return place(out, unplaced, ctxAt, modelAt, l.separator())
}

// partitionHooks splits the hooks into those named in the segments list, keyed by name, and the rest.
func partitionHooks(hooks []hook.Result, names []string) (byName map[string][]hook.Result, unplaced []hook.Result) {
	listed := map[string]bool{}
	for _, name := range names {
		listed[name] = true
	}
	byName = map[string][]hook.Result{}
	for _, h := range hooks {
		if h.Name != "" && listed[h.Name] {
			byName[h.Name] = append(byName[h.Name], h)
		} else {
			unplaced = append(unplaced, h)
		}
	}
	return byName, unplaced
}

func appendNamedHooks(out []layout.Item, hooks []hook.Result, builtin plugin.Segment, isBuiltin bool) []layout.Item {
	for _, h := range hooks {
		if h.Text == "" {
			continue
		}
		group := slotGroup(h.Slot)
		if isBuiltin {
			group = builtin.Group()
		}
		out = append(out, layout.Item{Group: group, Text: h.Text})
	}
	return out
}

func slotGroup(s hook.Slot) layout.Group {
	if s == hook.SlotCtx {
		return layout.Identity
	}
	return layout.Extra
}

// place inserts the hooks that the segments list does not name.
func place(out []layout.Item, hooks []hook.Result, ctxAt, modelAt int, sep string) []layout.Item {
	var inCtx []string
	var inUsage, atEnd []layout.Item
	for _, h := range hooks {
		if h.Text == "" {
			continue
		}
		switch h.Slot {
		case hook.SlotUsage:
			inUsage = append(inUsage, layout.Item{Group: layout.Extra, Text: h.Text})
		case hook.SlotEnd:
			atEnd = append(atEnd, layout.Item{Group: layout.Extra, Text: h.Text})
		default:
			inCtx = append(inCtx, h.Text)
		}
	}

	if len(inCtx) > 0 {
		// One unit, so that the hooks never split across rows.
		item := layout.Item{Group: layout.Identity, Text: strings.Join(inCtx, sep)}
		out = insert(out, afterIdentity(out, ctxAt, modelAt), item)
	}
	if len(inUsage) > 0 {
		out = insert(out, usageEnd(out), inUsage...)
	}
	return append(out, atEnd...)
}

// afterIdentity is the index to insert at to land after the context segment, else after the model, else after the last identity segment.
func afterIdentity(out []layout.Item, ctxAt, modelAt int) int {
	switch {
	case ctxAt >= 0:
		return ctxAt + 1
	case modelAt >= 0:
		return modelAt + 1
	}
	at := 0
	for i, it := range out {
		if it.Group == layout.Identity {
			at = i + 1
		}
	}
	return at
}

// usageEnd is the index after the last usage segment, or before the first rate segment when there is none.
func usageEnd(out []layout.Item) int {
	last, firstRate := -1, len(out)
	for i, it := range out {
		switch it.Group {
		case layout.Usage:
			last = i
		case layout.Rates:
			firstRate = min(firstRate, i)
		default:
			continue
		}
	}
	if last >= 0 {
		return last + 1
	}
	return firstRate
}

func insert(out []layout.Item, at int, add ...layout.Item) []layout.Item {
	out = append(out, add...)
	copy(out[at+len(add):], out[at:])
	copy(out[at:], add)
	return out
}
