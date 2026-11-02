package cache_test

import (
	"testing"
	"time"

	"github.com/PaddleHQ/claude-statusline/segments/cache"
	"github.com/PaddleHQ/claude-statusline/segments/cache/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func build(t *testing.T, input, creation, read int64, warn int) *cache.Cache {
	t.Helper()

	sess := mocks.NewSession(t)
	sess.EXPECT().LatestRequest().Return(input, creation, read).Maybe()
	sess.EXPECT().TranscriptFile().Return("").Maybe()
	cfg := mocks.NewConfig(t)
	cfg.EXPECT().Int("warn").Return(warn).Maybe()
	paint := mocks.NewPainter(t)
	paint.EXPECT().Text(mock.Anything).RunAndReturn(func(s string) string { return "<t>" + s + "</>" }).Maybe()
	paint.EXPECT().Bad(mock.Anything).RunAndReturn(func(s string) string { return "<b>" + s + "</>" }).Maybe()
	return cache.New(cache.Deps{Session: sess, Config: cfg, Paint: paint, Icons: plainIcons(t)})
}

func plainIcons(t *testing.T) *mocks.Icons {
	t.Helper()

	icons := mocks.NewIcons(t)
	icons.EXPECT().Pick(mock.Anything, mock.Anything).RunAndReturn(func(plain, _ string) string { return plain }).Maybe()
	return icons
}

func TestRender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		input, create, read int64
		warn                int
		want                string
	}{
		{"healthy", 10, 10, 80, 5, "<t>⧉ 80%</>"},
		{"at the threshold is not red", 95, 0, 5, 5, "<t>⧉ 5%</>"},
		{"below the threshold is red", 96, 0, 4, 5, "<b>⧉ 4%</>"},
		{"three percent is red", 50, 50, 4, 5, "<b>⧉ 3%</>"},
		{"integer division truncates", 1, 0, 2, 0, "<t>⧉ 66%</>"},
		{"custom threshold", 10, 10, 80, 90, "<b>⧉ 80%</>"},
		{"all reads is 100", 0, 0, 10, 5, "<t>⧉ 100%</>"},
		{"zero denominator hides the segment", 0, 0, 0, 5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, build(t, tt.input, tt.create, tt.read, tt.warn).Render())
		})
	}
}

func TestRenderExpiredCache(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		file    string
		ago     time.Duration
		ttl     time.Duration
		unknown bool
		noUsage bool
		want    string
	}{
		{"an expired cache is cold even with no usage reported", "/t.jsonl", 2 * time.Hour, time.Hour, false, true, "<b>⧉ cold</>"},
		{"a live cache with no usage reported is hidden", "/t.jsonl", time.Minute, time.Hour, false, true, ""},
		{"within the TTL shows the hit rate", "/t.jsonl", 59 * time.Minute, time.Hour, false, false, "<t>⧉ 80%</>"},
		{"at the TTL is cold", "/t.jsonl", time.Hour, time.Hour, false, false, "<b>⧉ cold</>"},
		{"a resumed session from yesterday is cold", "/t.jsonl", 24 * time.Hour, time.Hour, false, false, "<b>⧉ cold</>"},
		{"a five-minute cache goes cold sooner", "/t.jsonl", 6 * time.Minute, 5 * time.Minute, false, false, "<b>⧉ cold</>"},
		{"an unknown request time is never cold", "/t.jsonl", 0, time.Hour, true, false, "<t>⧉ 80%</>"},
		{"no transcript is never cold", "", 24 * time.Hour, time.Hour, false, false, "<t>⧉ 80%</>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			at := now.Add(-tt.ago)
			if tt.unknown {
				at = time.Time{}
			}
			sess := mocks.NewSession(t)
			input, creation, read := int64(10), int64(10), int64(80)
			if tt.noUsage {
				input, creation, read = 0, 0, 0
			}
			sess.EXPECT().LatestRequest().Return(input, creation, read).Maybe()
			sess.EXPECT().TranscriptFile().Return(tt.file)
			tr := mocks.NewTranscript(t)
			tr.EXPECT().LastRequest().Return(at, tt.ttl).Maybe()
			paint := mocks.NewPainter(t)
			paint.EXPECT().Text(mock.Anything).RunAndReturn(func(s string) string { return "<t>" + s + "</>" }).Maybe()
			paint.EXPECT().Bad(mock.Anything).RunAndReturn(func(s string) string { return "<b>" + s + "</>" }).Maybe()
			cfg := mocks.NewConfig(t)
			cfg.EXPECT().Int("warn").Return(5).Maybe()
			seg := cache.New(cache.Deps{
				Session:    sess,
				Transcript: func() cache.Transcript { return tr },
				Config:     cfg,
				Paint:      paint,
				Now:        now,
				Icons:      plainIcons(t),
			})
			assert.Equal(t, tt.want, seg.Render())
		})
	}
}

func TestNameAndKeys(t *testing.T) {
	t.Parallel()

	seg := build(t, 0, 0, 0, 5)
	assert.Equal(t, "cache", seg.Name())
	assert.Equal(t, []string{"warn"}, seg.Keys())
}

func TestHelp(t *testing.T) {
	t.Parallel()

	assert.Contains(t, build(t, 0, 0, 0, 12).Help(), "12%")
}

func TestCacheNerdIcon(t *testing.T) {
	t.Parallel()

	sess := mocks.NewSession(t)
	sess.EXPECT().LatestRequest().Return(int64(0), int64(0), int64(100))
	sess.EXPECT().TranscriptFile().Return("")
	cfg := mocks.NewConfig(t)
	cfg.EXPECT().Int("warn").Return(10)
	paint := mocks.NewPainter(t)
	paint.EXPECT().Text(mock.Anything).RunAndReturn(func(s string) string { return s })
	icons := mocks.NewIcons(t)
	icons.EXPECT().Pick("\u29C9 ", "\uF0E7 ").RunAndReturn(func(_, nerd string) string { return nerd })
	seg := cache.New(cache.Deps{Session: sess, Config: cfg, Paint: paint, Icons: icons})
	assert.Equal(t, "\uF0E7 100%", seg.Render())
}

func TestCacheColdNerdIcon(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	sess := mocks.NewSession(t)
	sess.EXPECT().TranscriptFile().Return("/t.jsonl")
	tr := mocks.NewTranscript(t)
	tr.EXPECT().LastRequest().Return(now.Add(-2*time.Hour), time.Hour)
	paint := mocks.NewPainter(t)
	paint.EXPECT().Bad(mock.Anything).RunAndReturn(func(s string) string { return s })
	icons := mocks.NewIcons(t)
	icons.EXPECT().Pick("⧉ ", " ").RunAndReturn(func(_, nerd string) string { return nerd })
	seg := cache.New(cache.Deps{
		Session:    sess,
		Transcript: func() cache.Transcript { return tr },
		Config:     mocks.NewConfig(t),
		Paint:      paint,
		Now:        now,
		Icons:      icons,
	})
	assert.Equal(t, " cold", seg.Render())
}

func TestHelpShowsTheChosenColdLabel(t *testing.T) {
	t.Parallel()

	cfg := mocks.NewConfig(t)
	cfg.EXPECT().Int("warn").Return(10)
	icons := mocks.NewIcons(t)
	icons.EXPECT().Pick("⧉ ", " ").RunAndReturn(func(_, nerd string) string { return nerd })
	seg := cache.New(cache.Deps{Session: mocks.NewSession(t), Config: cfg, Paint: mocks.NewPainter(t), Icons: icons})
	assert.Contains(t, seg.Help(), "a red  cold once")
}
