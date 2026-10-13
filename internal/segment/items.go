package segment

import (
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/hook"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
)

// items renders the configured segments in order, then places the hooks among them.
//
// A hook named in the segments list renders there, and replaces a built-in of the same name.
// Every other hook goes to its slot: after the context segment, after the usage group, or at the end.
func items(ctx *Context) []layout.Item {
	byName := map[string][]hook.Result{}
	var unplaced []hook.Result
	listed := map[string]bool{}
	names := strings.Fields(ctx.Config.String(keySegments))
	for _, name := range names {
		listed[name] = true
	}
	for _, h := range ctx.Hooks {
		if h.Name != "" && listed[h.Name] {
			byName[h.Name] = append(byName[h.Name], h)
		} else {
			unplaced = append(unplaced, h)
		}
	}

	var out []layout.Item
	ctxAt, modelAt := -1, -1
	for _, name := range names {
		builtin, isBuiltin := Lookup(name)
		if hooks := byName[name]; len(hooks) > 0 {
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
			continue
		}
		if !isBuiltin {
			continue
		}
		text := builtin.Render(ctx)
		if text == "" {
			continue
		}
		out = append(out, layout.Item{Group: builtin.Group(), Text: text})
		switch name {
		case "ctx":
			ctxAt = len(out) - 1
		case "model":
			modelAt = len(out) - 1
		}
	}
	return place(out, unplaced, ctxAt, modelAt, separator(ctx))
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
