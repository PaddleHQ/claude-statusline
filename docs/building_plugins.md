# Building plugins

A segment is a small plugin: one folder under [`segments/`](../segments) that draws one part of the line. This guide builds one step by step, using [`segments/model`](../segments/model) as the worked example. For what each setting does to users, see [Customising](customising.md); for the rest of the code base and its conventions, see [Contributing](../CONTRIBUTING.md).

## What a plugin is

A package under `segments/` provides one or more segments, which are the names users put in `segments`. `segments/rate` provides `rate5h` and `rate7d`, and `segments/diagnostics` provides `boot`, `skills`, `skills_stripped`, `agent_skills` and `mem`. A package is the right size when its segments share code or keys; otherwise give each its own.

The package owns its code, the defaults for its config keys, its README and its tests. It knows nothing about the rest of the program:

- **A plugin imports nothing from `internal/`.** It declares the small interfaces it needs, in its own package, and receives implementations through its constructor. `TestPluginsImportNothingFromInternal` in `segments/all_test.go` fails on any import of an `internal/` package in a plugin's non-test files.
- **Expensive dependencies are passed as functions**, so they are only paid for when the segment draws: `func() Transcript` reads a transcript file, and `func() Limits` reads and writes the shared rate cache. Call them from `Render`, only when you need them, never from the constructor. `TestLazyDependencies` in `segments/all_test.go` pins this.
- **A plugin's methods use only types from its own package, or builtin types.** The layout group is not its concern.
- **[`segments/all.go`](../segments/all.go) is the composition root**, the one file under `segments/` that may import `internal/`. It registers each segment's name, layout group and constructor, and hands the program's internal types (the payload, the theme, the config section, the transcript, the rate cache, the bar drawing) to the plugins. There are no adapter types: Go interfaces are satisfied structurally, so an internal type gains the methods a plugin's interface asks for, and the plugin and the internal package still do not import each other.

Plugins do not share helpers through a common package. Where two need the same behaviour it is injected: `ctx` and `rate` each declare a `Meter` that draws the bar, and `diagnostics` takes a `Tokens` function that `all.go` supplies from the `tokens` plugin.

## The worked example

### 1. Declare what you need

Write one interface per thing the segment depends on, with only the methods it calls, named for what the segment needs. Put a `go:generate` line above each, so that a mock is generated for the tests. These are `model`'s:

```go
//go:generate go run github.com/vektra/mockery/v2 --name=Session

// Session is what the segment needs to know about the session.
type Session interface {
	DisplayName() string
	ModelID() string
	EffortLevel() string
	TranscriptFile() string
}

//go:generate go run github.com/vektra/mockery/v2 --name=Transcript

// Transcript is what the segment needs from the session transcript.
type Transcript interface {
	LatestModel() string
}

//go:generate go run github.com/vektra/mockery/v2 --name=Config

// Config reads the segment's settings.
type Config interface {
	Bool(key string) bool
}

//go:generate go run github.com/vektra/mockery/v2 --name=Painter

// Painter colors text.
type Painter interface {
	Text(s string) string
}

// Deps is everything the segment is built from. Transcript is a function so that the transcript is only read when the segment needs it.
type Deps struct {
	Session    Session
	Transcript func() Transcript
	Config     Config
	Paint      Painter
}
```

The `go:generate` lines are already there in the source and look like this, with the interface name changed:

```go
//go:generate go run github.com/vektra/mockery/v2 --name=Session
```

Keep the interfaces small. A method the segment does not call does not belong in its interface, and a session question like `DisplayName()` is better than handing the segment a whole payload.

### 2. Collect them in `Deps`, and write `New`

```go
type Deps struct {
	Session    Session
	Transcript func() Transcript
	Config     Config
	Paint      Painter
}

type Model struct{ d Deps }

func New(d Deps) *Model { return &Model{d: d} }
```

`New` only stores what it is given. It must be cheap, because a segment is built whenever its name is in `segments`, even if a custom segment ends up replacing it. The config is already scoped to the segment, so `Config.Bool("effort")` reads `model.effort`.

### 3. Write the methods

Behaviour is a method on the type, with a named pointer receiver (see "Go style" in Contributing). A segment has four methods:

```go
func (m *Model) Name() string
func (m *Model) Help() string
func (m *Model) Keys() []string
func (m *Model) Render() string
```

`Help` is one or two plain sentences saying what the segment shows and what its colours mean, for the `help` command and `/statusline:help`. Build it from the segment's own config, so it reflects the user's settings: the `ctx` segment says "How full the context window is. Amber from 50%, red from 90%." with the configured `warn` and `bad`. A package with several segments gives each its own text.

`Keys` lists every config key the segment reads, relative to its section, including keys that a dependency such as a `Meter` reads on its behalf. `Render` returns the finished text, already painted through the `Painter`, or an empty string to hide the segment. A segment is hidden, not wrong, when its data is missing. A plugin never prints and never returns an error. `model` only asks for the transcript when the payload names one, so a session without a transcript pays nothing for it.

### 4. Give it defaults

`segments/<name>/defaults.conf` holds the plugin's keys, with full names and a comment above each:

```conf
# Show the effort level after the model name (true or false).
model.effort = true
```

Every key a segment reads must be in its `defaults.conf` with a comment, and every key in the file must be read by its segment, which `Keys()` declares. `TestKeysMatchDefaults` in `internal/statusline` checks both directions across every plugin. Key names are `<segment>.<setting>`, lower case, one dot. Settings that are not about a segment (`segments`, `separator`, `theme`, `hooks.*`, `snapshot.*`) stay in `internal/config/defaults.conf`. To show a segment by default, add its name to the `segments` default there too. A plugin with no settings has no `defaults.conf`. The plugin embeds its own file into an exported `Defaults` string (`//go:embed defaults.conf`), and `Defaults()` in `all.go` lists them.

User config is layered over the union of all the defaults, so a user's `model.effort = false` wins, and an invalid value falls back to the default.

### 5. Write its README

Every plugin has a short `README.md` (under about 30 lines): what it shows, in one sentence with an example; a table of its config keys and defaults, which must match its `defaults.conf` (`TestEveryPluginReadmeDocumentsItsDefaults` checks each row); what it depends on, in plain words; and the layout group `all.go` gives it. Copy [`segments/model/README.md`](../segments/model/README.md).

### 6. Register it in `all.go`

In `segments/all.go`, add your package's `Defaults` to `Defaults()`, then add one entry to `registrations`: the name, the layout group, and a constructor that passes the program's types to your interfaces:

```go
{"model", layout.Identity, func(cfg *config.Section) Segment {
	return model.New(model.Deps{
		Session:    &e.Payload,
		Transcript: func() model.Transcript { return e.info() },
		Config:     cfg,
		Paint:      &e.Theme,
	})
}},
```

`cfg` is the config scoped to the segment's name, and a `*config.Section` already has the `Bool`, `Int` and `String` methods the plugins ask for. The session is `payload.Payload`, the painter is `theme.Theme`, the transcript is `transcript.Info` and the rate limits are `ratecache.Limits`; when your interface needs a new question answered, add a method to the internal type. Name the method so that it does not collide with a field of that type (`Payload.Cwd` is why the session asks for `WorkingDir`). The transcript and rate-limit functions are memoised by `NewEnv`, so however many segments ask, the work happens once, and not at all if none does.

The layout group decides how the segment behaves when the line is too narrow for one row (see [Wrapping](customising.md#wrapping)):

| Group | Behaviour |
|---|---|
| `layout.Identity` | Packs greedily into the top rows: path, model, context. |
| `layout.Usage` | Sits flush left on the usage row: tokens, cache and the transcript diagnostics. |
| `layout.Rates` | Stays together with the other rate bars, lined up under the second identity segment. |
| `layout.Extra` | Takes a row of its own. This is the group of custom segments that are not in a built-in slot. |

There is no registration magic: a plugin that is not in `registrations` does not exist. The help command and the keys check read the same list. User [custom segments](customising.md#custom-segments) are placed among the built-ins by name and slot, and a hook named like a plugin replaces it. That needs nothing from the plugin.

### 7. Test it with mocks

Run `go generate ./...` to create the mocks. They are written to a `mocks/` folder inside the package, one file per interface, and are committed. CI regenerates them and fails if the committed ones differ, so rerun `go generate ./...` whenever you change an interface.

Tests are in an external `_test` package and use the typed expecter API. Nothing from `internal/` is needed:

```go
sess := mocks.NewSession(t)
sess.EXPECT().DisplayName().Return("Opus 4.7")
sess.EXPECT().EffortLevel().Return("high")
sess.EXPECT().TranscriptFile().Return("")

cfg := mocks.NewConfig(t)
cfg.EXPECT().Bool("effort").Return(true)

paint := mocks.NewPainter(t)
paint.EXPECT().Text("Opus 4.7 [high]").Return("<t>Opus 4.7 [high]</t>")

seg := model.New(model.Deps{Session: sess, Config: cfg, Paint: paint})
assert.Equal(t, "<t>Opus 4.7 [high]</t>", seg.Render())
```

A mock fails the test when an expected call does not happen, and when an unexpected one does, so a test that gives the segment no transcript function proves that it never asks for one. Use `.Maybe()` on a call that a particular case may or may not make. Cover the case where the data is missing, as well as the ones where it is there, and test that `Help` is not empty and mentions every threshold or label it claims to.

These tests are the plugin's own, and cover every edge of what it draws: the formatting, each threshold and colour, the missing data and each config variant. The feature files only show the overall line, so add one representative scenario for the segment under `features/` (see "Which layer does my test belong in?" in Contributing), and regenerate the documentation images if a `@screenshot` scenario changes.

### 8. Document it

Add the segment to the table in [Customising](customising.md), and its keys to the key table there; the table must match the union of the `defaults.conf` files. Describe what the segment shows and when it is hidden.
