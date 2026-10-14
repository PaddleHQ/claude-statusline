# Building plugins

A segment is a small plugin: one folder under [`segments/`](../segments) that draws one part of the line. This guide builds one step by step, using [`segments/model`](../segments/model) as the worked example. For what each setting does to users, see [Customising](customising.md); for the rest of the code base and its conventions, see [Contributing](../CONTRIBUTING.md).

## What a plugin is

A package under `segments/` provides one or more segments, which are the names users put in `segments`. `segments/rate` provides `rate5h` and `rate7d`, and `segments/diagnostics` provides `boot`, `skills`, `skills_stripped`, `agent_skills` and `mem`. A package is the right size when its segments share code or keys; otherwise give each its own.

The package owns everything about its segments: the code, the defaults for its config keys, and the tests. The only places outside it that change are the one line in `segments/all.go` that lists it and the documentation.

## The interface

The shared types live in [`internal/plugin`](../internal/plugin).

```go
// Segment renders one part of the line.
type Segment interface {
	Name() string
	// Group says which part of the line the segment belongs to when it wraps.
	Group() layout.Group
	// Keys lists every config key Render reads, relative to the segment's section, so a test can check them against defaults.conf.
	Keys() []string
	// Help says in a sentence or two what the segment shows and what its colors mean,
	// using the user's current settings for any threshold or label.
	Help() string
	// Render returns the segment's text, or "" to hide it.
	Render(*Session) string
}

// Session is what one render is about: the payload Claude Code sent.
type Session struct {
	Payload payload.Payload
}
```

`Help` is one or two plain sentences saying what the segment shows and what its colours mean, for the `help` command and `/statusline:help`. Build it from the segment's own config, so it reflects the user's settings: the `ctx` segment says "How full the context window is. Amber from 50%, red from 90%." with the configured `warn` and `bad`. A package with several segments gives each its own text. Test that it is not empty and that it mentions every threshold or label it claims to.

`Render` returns the finished text, already painted with the theme, or an empty string to hide the segment. A segment is hidden, not wrong, when its data is missing. Plugin code never prints and never returns an error.

## What a plugin may use

A segment is built with `Providers`, the shared, read-only data every plugin may use. They are per render, and they are the only way to reach the outside world.

| Provider | What it gives |
|---|---|
| `Theme` | The colours and glyphs; paint text with `p.Theme.Paint(p.Theme.Text, s)`. |
| `Config` | A view of the config scoped to the segment, so `Config.Bool("effort")` reads `model.effort`. |
| `Now()` | The time of the render. |
| `Columns`, `Home` | The terminal width and the home directory. |
| `Separator()` | The painted separator glyph, for segments that join several parts. |
| `BarWidth()` | The bar width for this terminal, from the segment's `width`, `width_medium` and `width_narrow` keys. |
| `Transcript()` | What the session transcript revealed. **Lazy.** |
| `RateLimits()` | The rate-limit readings, merged with the shared cache. **Lazy.** |

The last two are expensive: `Transcript()` reads a file, and `RateLimits()` reads and writes the shared cache. They do nothing until a segment calls them, and then do the work once for the whole render, however many segments ask. If no listed segment asks, the work never happens. So call them from `Render`, only when you need them, and never from the constructor. `TestLazyProviders` in `segments/all_test.go` pins this.

The scoping is by segment name. A package that provides several segments gets each one its own section, so `rate5h` reads `warn` and gets `rate5h.warn`, and `rate7d` gets `rate7d.warn`.

## Step by step

### 1. Create the package

Make `segments/<name>/` with a `<name>.go`. The package name is the folder name. It exports a `Plugin()` function, which says which segments it provides and the defaults they use, and a constructor per segment:

```go
//go:embed defaults.conf
var defaults string

// Plugin is the model plugin.
func Plugin() *plugin.Plugin {
	return &plugin.Plugin{Defaults: defaults, Segments: []plugin.Entry{{Name: name, New: New}}}
}

// New builds the model segment.
func New(p *plugin.Providers) plugin.Segment { return &model{p: p} }
```

The constructor receives the providers scoped to the segment and keeps them. It must be cheap, because a segment is built whenever its name is in `segments`, even if a custom segment ends up replacing it.

### 2. Write the segment

Behaviour is a method on the type, with a named pointer receiver (see "Go style" in Contributing).

```go
type model struct{ p *plugin.Providers }

func (m *model) Name() string { return name }

func (m *model) Group() layout.Group { return layout.Identity }

func (m *model) Help() string {
	help := "The model in use"
	if m.p.Config.Bool("effort") {
		help += " and its effort level, as in Opus 4.7 [high]"
	}
	return help + ". It reads Haiku (skill), for example, while a skill or subagent runs a different model than the session's."
}

func (m *model) Keys() []string { return []string{"effort"} }

func (m *model) Render(s *plugin.Session) string {
	th := m.p.Theme
	display := s.Payload.Model.DisplayName
	level := s.Payload.Effort.Level
	// A skill or subagent can run a different model than the session's, which the payload does not reflect.
	if s.Payload.TranscriptPath != "" {
		if actual, ok := family(m.p.Transcript().ActualModel); ok {
			if session, _ := family(s.Payload.Model.ID); actual.key != session.key {
				display, level = actual.label+" (skill)", ""
			}
		}
	}
	if display == "" {
		return ""
	}
	if level != "" && m.p.Config.Bool("effort") {
		display += " [" + level + "]"
	}
	return th.Paint(th.Text, display)
}
```

Two things to notice. The model segment only asks for the transcript when the payload names one, so a session without a transcript pays nothing for it. And it returns `""` when there is no model name.

### 3. Choose a layout group

`Group()` decides how the segment behaves when the line is too narrow for one row (see [Wrapping](customising.md#wrapping)):

| Group | Behaviour |
|---|---|
| `layout.Identity` | Packs greedily into the top rows: path, model, context. |
| `layout.Usage` | Sits flush left on the usage row: tokens, cache and the transcript diagnostics. |
| `layout.Rates` | Stays together with the other rate bars, lined up under the second identity segment. |
| `layout.Extra` | Takes a row of its own. This is the group of custom segments that are not in a built-in slot. |

User [custom segments](customising.md#custom-segments) are placed among the built-ins by name and slot, and a hook named like a plugin replaces it. That needs nothing from the plugin.

### 4. Give it defaults

`segments/<name>/defaults.conf` is embedded into the binary and holds the plugin's keys, with full names and a comment above each:

```conf
# Show the effort level after the model name (true or false).
model.effort = true
```

The key rule: every key a segment reads must be in its `defaults.conf` with a comment, and every key in the file must be read by its segment, which `Keys()` declares. `TestKeysMatchDefaults` in `internal/statusline` checks both directions across every plugin, so a setting cannot ship undocumented and a dead one cannot linger. Key names are `<segment>.<setting>`, lower case, one dot. Settings that are not about a segment (`segments`, `separator`, `theme`, `hooks.*`, `snapshot.*`) stay in `internal/config/defaults.conf`.

To show a segment by default, add its name to the `segments` default there too.

User config is layered over the union of all the defaults, so a user's `model.effort = false` wins, and an invalid value falls back to the default.

### 5. Test it

Tests are in an external `_test` package, as everywhere else. Build the providers from `plugin.Sources`, feeding the data you want, and render a `Session`:

```go
func render(json, cfg string, th theme.Theme, info transcript.Info) string {
	providers := plugin.NewProviders(&plugin.Sources{
		Config:     config.New(cfg, model.Plugin().Defaults),
		Theme:      th,
		Transcript: func() transcript.Info { return info },
	})
	seg := model.New(providers.For("model"))
	return seg.Render(&plugin.Session{Payload: payload.Parse([]byte(json))})
}
```

Cover the case where the data is missing, as well as the ones where it is there. Add at least one scenario under `features/` (see "Feature tests" in Contributing), and regenerate the documentation images if a `@screenshot` scenario changes.

### 6. List it

Add one line to `segments/all.go`:

```go
func All() []*plugin.Plugin {
	return []*plugin.Plugin{
		model.Plugin(),
		// ...
	}
}
```

There is no registration magic: a plugin that is not in this list does not exist. The list also supplies the defaults and the keys check, so nothing else needs wiring.

### 7. Document it

Add the segment to the table in [Customising](customising.md), and its keys to the key table there; the table must match the union of the `defaults.conf` files. Describe what the segment shows and when it is hidden.
