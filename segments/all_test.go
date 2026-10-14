package segments_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
	"github.com/PaddleHQ/claude-statusline/internal/render"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/internal/transcript"
	"github.com/PaddleHQ/claude-statusline/segments"

	"github.com/stretchr/testify/assert"
)

func TestEntryNamesMatchTheirSegments(t *testing.T) {
	t.Parallel()

	providers := plugin.NewProviders(&plugin.Sources{Config: config.New("")})
	seen := map[string]bool{}
	for _, pl := range segments.All() {
		for _, e := range pl.Segments {
			assert.Equal(t, e.Name, e.New(providers).Name(), "segment listed as %q", e.Name)
			assert.False(t, seen[e.Name], "%q is provided twice", e.Name)
			seen[e.Name] = true
		}
	}
}

func TestEverySegmentHasHelp(t *testing.T) {
	t.Parallel()

	providers := plugin.NewProviders(&plugin.Sources{Config: config.New("", segments.Defaults()...)})
	for _, pl := range segments.All() {
		for _, e := range pl.Segments {
			assert.NotEmpty(t, e.New(providers.For(e.Name)).Help(), e.Name)
		}
	}
}

func TestLazyProviders(t *testing.T) {
	t.Parallel()

	const (
		bare     = `{"model":{"display_name":"Opus"}}`
		withPath = `{"model":{"display_name":"Opus"},"transcript_path":"/t.jsonl"}`
	)
	tests := []struct {
		name           string
		cfg            string
		payload        string
		wantTranscript int
		wantRates      int
	}{
		{"model without a transcript asks for neither", "segments = model", bare, 0, 0},
		{"model with a transcript reads it but never the rate cache", "segments = model", withPath, 1, 0},
		{"model twice still reads the transcript once", "segments = model model", withPath, 1, 0},
		{"segments that need neither never trigger either", "segments = path tokens cache ctx", withPath, 0, 0},
		{"a rate bar syncs the cache and does not read the transcript", "segments = rate5h rate7d", withPath, 0, 1},
		{"a diagnostic reads the transcript and does not sync the cache", "segments = boot mem", withPath, 1, 0},
		{"unlisted segments are never asked", "segments = path", withPath, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var transcriptReads, rateSyncs int
			line := &render.Line{
				Plugins: segments.All(),
				Providers: plugin.NewProviders(&plugin.Sources{
					Config:     config.New(tt.cfg, segments.Defaults()...),
					Theme:      theme.Load("classic", "", ""),
					Transcript: func() transcript.Info { transcriptReads++; return transcript.Info{} },
					RateLimits: func() ratecache.Limits { rateSyncs++; return ratecache.Limits{} },
				}),
			}
			line.Session = &plugin.Session{Payload: payload.Parse([]byte(tt.payload))}
			line.Render()
			assert.Equal(t, tt.wantTranscript, transcriptReads, "transcript reads")
			assert.Equal(t, tt.wantRates, rateSyncs, "rate cache syncs")
		})
	}
}
