package plugin_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
	"github.com/PaddleHQ/claude-statusline/internal/transcript"

	"github.com/stretchr/testify/assert"
)

func TestProvidersComputeLazilyAndOnce(t *testing.T) {
	t.Parallel()

	var transcriptReads, rateSyncs int
	p := plugin.NewProviders(&plugin.Sources{
		Config:     config.New(""),
		Transcript: func() transcript.Info { transcriptReads++; return transcript.Info{Boot: 7} },
		RateLimits: func() ratecache.Limits { rateSyncs++; return ratecache.Limits{} },
	})
	scoped := p.For("model")

	assert.Zero(t, transcriptReads+rateSyncs, "nothing is computed until a segment asks")

	assert.EqualValues(t, 7, scoped.Transcript().Boot)
	assert.EqualValues(t, 7, p.Transcript().Boot)
	assert.Equal(t, 1, transcriptReads, "the transcript is read once, however many scopes ask")
	assert.Zero(t, rateSyncs, "asking for the transcript does not sync the rate cache")

	scoped.RateLimits()
	p.RateLimits()
	assert.Equal(t, 1, rateSyncs)
}

func TestProvidersWithoutSourcesGiveZeroValues(t *testing.T) {
	t.Parallel()

	p := plugin.NewProviders(&plugin.Sources{Config: config.New("")})
	assert.Equal(t, transcript.Info{}, p.Transcript())
	assert.Equal(t, ratecache.Limits{}, p.RateLimits())
}

func TestForScopesTheConfig(t *testing.T) {
	t.Parallel()

	p := plugin.NewProviders(&plugin.Sources{Config: config.New("a.size = 3\nb.size = 4")})
	assert.Equal(t, 3, p.For("a").Config.Int("size"))
	assert.Equal(t, 4, p.For("b").Config.Int("size"))
	assert.Equal(t, 3, p.For("a").For("a").Config.Int("size"))
	assert.Empty(t, p.Config.String("size"), "the original keeps its own scope")
}

func TestBarWidthFollowsTheTerminal(t *testing.T) {
	t.Parallel()

	cfg := config.New("x.width = 20\nx.width_medium = 14\nx.width_narrow = 10")
	for columns, want := range map[int]int{200: 20, 140: 14, 100: 10} {
		p := plugin.NewProviders(&plugin.Sources{Config: cfg, Columns: columns}).For("x")
		assert.Equal(t, want, p.BarWidth(), "at %d columns", columns)
	}
}
