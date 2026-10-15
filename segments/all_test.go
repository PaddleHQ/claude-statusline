package segments_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
	"github.com/PaddleHQ/claude-statusline/internal/render"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/internal/transcript"
	"github.com/PaddleHQ/claude-statusline/segments"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newEnv(cfg, payloadJSON string) *segments.Env {
	return segments.NewEnv(segments.Env{
		Payload: payload.Parse([]byte(payloadJSON)),
		Config:  config.New(cfg, segments.Defaults()...),
		Theme:   theme.Load("classic", "", ""),
		Columns: 999,
		Home:    "/home/me",
	})
}

func TestEveryRegisteredNameIsUniqueAndFindable(t *testing.T) {
	t.Parallel()

	env := newEnv("", `{}`)
	seen := map[string]bool{}
	for _, seg := range env.All() {
		assert.False(t, seen[seg.Name()], "%q is provided twice", seg.Name())
		seen[seg.Name()] = true
		found, _, ok := env.Lookup(seg.Name())
		require.True(t, ok, seg.Name())
		assert.Equal(t, seg.Name(), found.Name())
	}
	assert.Len(t, seen, 12)
	_, _, ok := env.Lookup("bogus")
	assert.False(t, ok)
}

func TestEverySegmentHasHelp(t *testing.T) {
	t.Parallel()

	for _, seg := range newEnv("", `{}`).All() {
		assert.NotEmpty(t, seg.Help(), seg.Name())
	}
}

func TestLayoutGroups(t *testing.T) {
	t.Parallel()

	want := map[string]layout.Group{
		"path": layout.Identity, "model": layout.Identity, "ctx": layout.Identity,
		"tokens": layout.Usage, "cache": layout.Usage,
		"boot": layout.Usage, "skills": layout.Usage, "skills_stripped": layout.Usage, "agent_skills": layout.Usage, "mem": layout.Usage,
		"rate5h": layout.Rates, "rate7d": layout.Rates,
	}
	env := newEnv("", `{}`)
	for name, group := range want {
		_, got, ok := env.Lookup(name)
		require.True(t, ok, name)
		assert.Equal(t, group, got, name)
	}
}

func TestPathBranchIsSanitized(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "hostile")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref: refs/heads/a\x1b[31mb\n"), 0o600))

	env := segments.NewEnv(segments.Env{
		Payload: payload.Parse([]byte(`{"cwd":"` + dir + `"}`)),
		Config:  config.New("", segments.Defaults()...),
	})
	seg, _, ok := env.Lookup("path")
	require.True(t, ok)
	assert.Equal(t, dir+" (a[31mb)", seg.Render())
}

func TestLazyDependencies(t *testing.T) {
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
			cfg := config.New(tt.cfg, segments.Defaults()...)
			env := segments.NewEnv(segments.Env{
				Payload:    payload.Parse([]byte(tt.payload)),
				Config:     cfg,
				Theme:      theme.Load("classic", "", ""),
				Transcript: func() transcript.Info { transcriptReads++; return transcript.Info{} },
				RateLimits: func() ratecache.Limits { rateSyncs++; return ratecache.Limits{} },
			})
			env.All()
			require.Zero(t, transcriptReads+rateSyncs, "building the segments asks for nothing")
			line := &render.Line{
				Names: strings.Fields(cfg.String("segments")),
				Lookup: func(name string) (render.Segment, layout.Group, bool) {
					seg, group, ok := env.Lookup(name)
					if !ok {
						return nil, 0, false
					}
					return seg, group, true
				},
				Separator: env.Separator(),
			}
			line.Render()
			assert.Equal(t, tt.wantTranscript, transcriptReads, "transcript reads")
			assert.Equal(t, tt.wantRates, rateSyncs, "rate cache syncs")
		})
	}
}

func pluginDirs(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	require.NotEmpty(t, dirs)
	return dirs
}

func TestPluginsImportNothingFromInternal(t *testing.T) {
	t.Parallel()

	for _, dir := range pluginDirs(t) {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			file, perr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			require.NoError(t, perr, path)
			for _, imp := range file.Imports {
				assert.NotContains(t, imp.Path.Value, "/internal/", "%s imports %s", path, imp.Path.Value)
			}
			return nil
		})
		require.NoError(t, err)
	}
}

func TestEveryPluginReadmeDocumentsItsDefaults(t *testing.T) {
	t.Parallel()

	for _, dir := range pluginDirs(t) {
		readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
		require.NoError(t, err, "%s needs a README.md", dir)
		defaults, err := os.ReadFile(filepath.Join(dir, "defaults.conf"))
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)
		for line := range strings.SplitSeq(string(defaults), "\n") {
			key, value, ok := strings.Cut(line, "=")
			if !ok || strings.HasPrefix(line, "#") {
				continue
			}
			key, value = strings.TrimSpace(key), strings.TrimSpace(value)
			want := "`" + value + "`"
			if value == "" {
				want = "empty"
			}
			row := "| `" + key + "` | " + want + " |"
			assert.Contains(t, string(readme), row, "%s/README.md must list %s with its default", dir, key)
		}
	}
}
