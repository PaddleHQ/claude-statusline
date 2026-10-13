package hook_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/PaddleHQ/claude-statusline/internal/hook"
)

var epoch = time.Unix(1760000000, 0)

func script(t *testing.T, dir, name, body string, perm os.FileMode) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o700))
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(body), perm))
	require.NoError(t, os.Chmod(path, perm))
	return path
}

func options(cache string) hook.Options {
	return hook.Options{CacheDir: cache, Session: "s1", Now: epoch, Timeout: 5 * time.Second, DefaultTTL: 60 * time.Second}
}

func runOne(t *testing.T, hooks []hook.Hook, o hook.Options) string {
	t.Helper()
	got := hook.Run(hooks, o)
	require.Len(t, got, 1)
	return got[0].Text
}

func TestDiscover(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	script(t, dir, "20-b.sh", "#!/bin/sh\n", 0o700)
	script(t, dir, "10-a.sh", "#!/bin/sh\n", 0o700)
	script(t, dir, "notexec.sh", "#!/bin/sh\n", 0o600)
	script(t, dir, ".hidden", "#!/bin/sh\n", 0o700)
	require.NoError(t, os.Mkdir(filepath.Join(dir, "subdir"), 0o700))

	var files []string
	for _, h := range hook.Discover(dir) {
		files = append(files, h.File)
	}
	assert.Equal(t, []string{"10-a.sh", "20-b.sh"}, files, "regular executable files in filename order")
	assert.Empty(t, hook.Discover(filepath.Join(dir, "missing")))
}

func TestDiscoverFollowsSymlinks(t *testing.T) {
	t.Parallel()

	dir, other := t.TempDir(), t.TempDir()
	target := script(t, other, "real.sh", "#!/bin/sh\n", 0o700)
	require.NoError(t, os.Symlink(target, filepath.Join(dir, "link.sh")))

	hooks := hook.Discover(dir)
	require.Len(t, hooks, 1)
	assert.Equal(t, "link", hooks[0].Name)
}

func TestHookName(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"10-backup.sh":  "backup",
		"backup.sh":     "backup",
		"backup":        "backup",
		"5-a-b.py":      "a-b",
		"007-x.sh":      "x",
		"10backup.sh":   "10backup",
		"a.b.sh":        "a.b",
		"10-10-more.sh": "10-more",
	}
	for file, want := range tests {
		t.Run(file, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			script(t, dir, file, "#!/bin/sh\n", 0o700)
			hooks := hook.Discover(dir)
			require.Len(t, hooks, 1)
			assert.Equal(t, want, hooks[0].Name)
		})
	}
}

func TestHeaderDirectives(t *testing.T) {
	t.Parallel()

	filler := strings.Repeat("#\n", 19)
	tests := []struct {
		name     string
		body     string
		wantSlot hook.Slot
		wantTTL  time.Duration
	}{
		{"defaults", "#!/bin/sh\n", hook.SlotCtx, -1},
		{"usage slot", "#!/bin/sh\n# statusline-slot: usage\n", hook.SlotUsage, -1},
		{"end slot", "#!/bin/sh\n#statusline-slot:end\n", hook.SlotEnd, -1},
		{"ctx slot", "#!/bin/sh\n# statusline-slot: ctx\n", hook.SlotCtx, -1},
		{"unknown slot falls back to ctx", "#!/bin/sh\n# statusline-slot: bogus\n", hook.SlotCtx, -1},
		{"capitalised slot is not a slot", "#!/bin/sh\n# statusline-slot: Usage\n", hook.SlotCtx, -1},
		{"slot with trailing words is not a slot", "#!/bin/sh\n# statusline-slot: usage now\n", hook.SlotCtx, -1},
		{"ttl", "#!/bin/sh\n# statusline-ttl: 300\n", hook.SlotCtx, 300 * time.Second},
		{"ttl with trailing text", "#!/bin/sh\n# statusline-ttl: 30 seconds\n", hook.SlotCtx, 30 * time.Second},
		{"ttl zero", "#!/bin/sh\n# statusline-ttl: 0\n", hook.SlotCtx, 0},
		{"invalid ttl", "#!/bin/sh\n# statusline-ttl: soon\n", hook.SlotCtx, -1},
		{"first directive wins", "#!/bin/sh\n# statusline-slot: end\n# statusline-slot: usage\n# statusline-ttl: 5\n# statusline-ttl: 9\n", hook.SlotEnd, 5 * time.Second},
		{"both directives", "#!/bin/sh\n# statusline-slot: usage\n# statusline-ttl: 300\n", hook.SlotUsage, 300 * time.Second},
		{"line 20 is read", "#!/bin/sh\n" + filler[:36] + "# statusline-slot: end\n", hook.SlotEnd, -1},
		{"line 21 is ignored", "#!/bin/sh\n" + filler + "# statusline-slot: end\n# statusline-ttl: 5\n", hook.SlotCtx, -1},
		{"not a comment", "#!/bin/sh\necho '# statusline-slot: end'\nstatusline-slot: end\n", hook.SlotCtx, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			script(t, dir, "h.sh", tt.body, 0o700)
			hooks := hook.Discover(dir)
			require.Len(t, hooks, 1)
			assert.Equal(t, tt.wantSlot, hooks[0].Slot)
			assert.Equal(t, tt.wantTTL, hooks[0].TTL)
		})
	}
}

func TestRunContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{"output is shown", "printf shown", "shown"},
		{"first line only", "printf 'one\\ntwo\\n'", "one"},
		{"no output shows nothing", "exit 0", ""},
		{"non-zero shows nothing without a cache", "echo broken; exit 3", ""},
		{"stdin carries the session JSON", `cat`, `{"a":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			script(t, dir, "h.sh", "#!/bin/sh\n"+tt.body+"\n", 0o700)
			o := options(t.TempDir())
			o.Input = []byte(`{"a":1}`)
			assert.Equal(t, tt.want, runOne(t, hook.Discover(dir), o))
		})
	}
}

func TestRunWithoutCacheDirectoryAlwaysRuns(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	counter := filepath.Join(t.TempDir(), "runs")
	script(t, dir, "h.sh", "#!/bin/sh\necho x >> "+counter+"\nprintf hi\n", 0o700)
	hooks := hook.Discover(dir)

	assert.Equal(t, "hi", runOne(t, hooks, options("")))
	assert.Equal(t, "hi", runOne(t, hooks, options("")))
	data, err := os.ReadFile(counter)
	require.NoError(t, err)
	assert.Equal(t, 2, strings.Count(string(data), "x"))
}

func TestRunCaching(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T, header string) (hooks []hook.Hook, cache, counter string) {
		t.Helper()
		dir, cache := t.TempDir(), t.TempDir()
		counter = filepath.Join(t.TempDir(), "runs")
		script(t, dir, "h.sh", "#!/bin/sh\n"+header+"echo x >> "+counter+"\nprintf fresh\n", 0o700)
		return hook.Discover(dir), cache, counter
	}
	runs := func(t *testing.T, counter string) int {
		t.Helper()
		data, err := os.ReadFile(counter)
		if os.IsNotExist(err) {
			return 0
		}
		require.NoError(t, err)
		return strings.Count(string(data), "x")
	}
	at := func(o hook.Options, d time.Duration) hook.Options {
		o.Now = o.Now.Add(d)
		return o
	}

	t.Run("not rerun inside the default TTL", func(t *testing.T) {
		t.Parallel()
		hooks, cache, counter := setup(t, "")
		o := options(cache)
		assert.Equal(t, "fresh", runOne(t, hooks, o))
		assert.Equal(t, "fresh", runOne(t, hooks, at(o, 59*time.Second)))
		assert.Equal(t, 1, runs(t, counter))
		assert.Equal(t, "fresh", runOne(t, hooks, at(o, 60*time.Second)))
		assert.Equal(t, 2, runs(t, counter), "due again once the TTL has passed")
	})

	t.Run("a TTL directive overrides the default", func(t *testing.T) {
		t.Parallel()
		hooks, cache, counter := setup(t, "# statusline-ttl: 300\n")
		o := options(cache)
		runOne(t, hooks, o)
		runOne(t, hooks, at(o, 299*time.Second))
		assert.Equal(t, 1, runs(t, counter))
		runOne(t, hooks, at(o, 300*time.Second))
		assert.Equal(t, 2, runs(t, counter))
	})

	t.Run("a zero TTL runs every time", func(t *testing.T) {
		t.Parallel()
		hooks, cache, counter := setup(t, "# statusline-ttl: 0\n")
		runOne(t, hooks, options(cache))
		runOne(t, hooks, options(cache))
		assert.Equal(t, 2, runs(t, counter))
	})

	t.Run("sessions do not share results", func(t *testing.T) {
		t.Parallel()
		hooks, cache, counter := setup(t, "")
		o := options(cache)
		runOne(t, hooks, o)
		o.Session = "s2"
		runOne(t, hooks, o)
		assert.Equal(t, 2, runs(t, counter))
		o.Session = "../escape"
		runOne(t, hooks, o)
		o.Session = ""
		runOne(t, hooks, o)
		assert.Equal(t, 3, runs(t, counter), "unsafe and missing ids share one directory")
		assert.DirExists(t, filepath.Join(cache, "shared"))
	})

	t.Run("a corrupt cache entry is treated as missing", func(t *testing.T) {
		t.Parallel()
		hooks, cache, _ := setup(t, "")
		require.NoError(t, os.MkdirAll(filepath.Join(cache, "s1"), 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(cache, "s1", "h.sh"), []byte("garbage\nold\n"), 0o600))
		assert.Equal(t, "fresh", runOne(t, hooks, options(cache)))
	})
}

func seed(t *testing.T, cache, session, file string, ts time.Time, value string) {
	t.Helper()
	dir := filepath.Join(cache, session)
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, file), []byte(strconv.FormatInt(ts.Unix(), 10)+"\n"+value+"\n"), 0o600))
}

func TestRunKeepsLastValue(t *testing.T) {
	t.Parallel()

	expired := epoch.Add(-time.Hour)
	tests := []struct {
		name    string
		body    string
		timeout time.Duration
		want    string
	}{
		{"non-zero exit keeps the last value", "echo ignored; exit 1", 5 * time.Second, "stale"},
		{"timeout keeps the last value", "sleep 5; echo late", 300 * time.Millisecond, "stale"},
		{"exit 0 with no output clears it", "exit 0", 5 * time.Second, ""},
		{"exit 0 with output replaces it", "printf new", 5 * time.Second, "new"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, cache := t.TempDir(), t.TempDir()
			script(t, dir, "h.sh", "#!/bin/sh\n"+tt.body+"\n", 0o700)
			seed(t, cache, "s1", "h.sh", expired, "stale")
			o := options(cache)
			o.Timeout = tt.timeout
			assert.Equal(t, tt.want, runOne(t, hook.Discover(dir), o))
		})
	}
}

func TestRunBacksOffAfterFailure(t *testing.T) {
	t.Parallel()

	dir, cache := t.TempDir(), t.TempDir()
	counter := filepath.Join(t.TempDir(), "runs")
	script(t, dir, "h.sh", "#!/bin/sh\necho x >> "+counter+"\nexit 1\n", 0o700)
	seed(t, cache, "s1", "h.sh", epoch.Add(-time.Hour), "stale")
	hooks := hook.Discover(dir)

	assert.Equal(t, "stale", runOne(t, hooks, options(cache)))
	assert.Equal(t, "stale", runOne(t, hooks, options(cache)))
	data, err := os.ReadFile(counter)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(data), "x"), "a failed run still waits a TTL")
}

func TestRunTimeoutKillsGrandchildren(t *testing.T) {
	t.Parallel()

	dir, cache := t.TempDir(), t.TempDir()
	marker := filepath.Join(t.TempDir(), "survived")
	script(t, dir, "h.sh", "#!/bin/sh\n(sleep 1; echo alive > "+marker+") &\nsleep 30\n", 0o700)
	o := options(cache)
	o.Timeout = 200 * time.Millisecond

	start := time.Now()
	assert.Empty(t, runOne(t, hook.Discover(dir), o))
	assert.Less(t, time.Since(start), 2*time.Second, "the run does not wait for the hook")

	time.Sleep(1500 * time.Millisecond)
	assert.NoFileExists(t, marker, "the background child was killed with the hook")
}

func TestRunIsConcurrent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	meet := t.TempDir()
	// Each hook waits for the other to start, so only concurrent execution prints both letters.
	rendezvous := func(self, other string) string {
		return "#!/bin/sh\n" +
			"touch '" + filepath.Join(meet, self) + "'\n" +
			"i=0\n" +
			"while [ ! -e '" + filepath.Join(meet, other) + "' ]; do\n" +
			"  i=$((i + 1)); [ \"$i\" -gt 80 ] && { printf late; exit 0; }\n" +
			"  sleep 0.05\n" +
			"done\n" +
			"printf " + self + "\n"
	}
	script(t, dir, "10-a.sh", rendezvous("a", "b"), 0o700)
	script(t, dir, "20-b.sh", rendezvous("b", "a"), 0o700)

	got := hook.Run(hook.Discover(dir), options(t.TempDir()))

	require.Len(t, got, 2)
	assert.Equal(t, "a", got[0].Text, "results stay in filename order and a saw b running")
	assert.Equal(t, "b", got[1].Text)
}

func TestRunSanitisesOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{"colour is kept", `printf '\033[32mok\033[0m'`, "\x1b[32mok\x1b[0m"},
		{"compound colour is kept", `printf '\033[1;38;5;33mok\033[0m'`, "\x1b[1;38;5;33mok\x1b[0m"},
		{"cursor movement is stripped", `printf 'a\033[2Ab\033[Hc\033[2Jd'`, "a[2Ab[Hc[2Jd"},
		{"other escapes are stripped", `printf 'a\033]0;title\007b\033cd'`, "a]0;titlebcd"},
		{"carriage return and tab are stripped", `printf 'a\rb\tc'`, "abc"},
		{"C1 controls are stripped", `printf 'a\302\233b\302\205c'`, "abc"},
		{"bell and backspace are stripped", `printf 'a\007b\010c'`, "abc"},
		{"unicode is kept", `printf '⇡ ok'`, "⇡ ok"},
		{"blank output is nothing", `printf '   '`, ""},
		{"lone escape at the end", `printf 'a\033'`, "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			script(t, dir, "h.sh", "#!/bin/sh\n"+tt.body+"\n", 0o700)
			assert.Equal(t, tt.want, runOne(t, hook.Discover(dir), options(t.TempDir())))
		})
	}
}

func TestRunSanitisesCachedValues(t *testing.T) {
	t.Parallel()

	dir, cache := t.TempDir(), t.TempDir()
	script(t, dir, "h.sh", "#!/bin/sh\nexit 1\n", 0o700)
	seed(t, cache, "s1", "h.sh", epoch, "a\x1b[2Jb")
	assert.Equal(t, "a[2Jb", runOne(t, hook.Discover(dir), options(cache)))
}

func TestRunSweepsOldSessions(t *testing.T) {
	t.Parallel()

	dir, cache := t.TempDir(), t.TempDir()
	script(t, dir, "h.sh", "#!/bin/sh\nprintf x\n", 0o700)
	old, recent := filepath.Join(cache, "old"), filepath.Join(cache, "recent")
	require.NoError(t, os.MkdirAll(old, 0o700))
	require.NoError(t, os.MkdirAll(recent, 0o700))
	// Directory times come from the real clock, so judge them against it.
	o := options(cache)
	o.Now = time.Now()
	require.NoError(t, os.Chtimes(old, o.Now, o.Now.Add(-8*24*time.Hour)))
	require.NoError(t, os.Chtimes(recent, o.Now, o.Now.Add(-6*24*time.Hour)))

	runOne(t, hook.Discover(dir), o)

	assert.NoDirExists(t, old)
	assert.DirExists(t, recent)
	assert.DirExists(t, filepath.Join(cache, "s1"))

	require.NoError(t, os.Chtimes(recent, o.Now, o.Now.Add(-30*24*time.Hour)))
	runOne(t, hook.Discover(dir), o)
	assert.DirExists(t, recent, "an existing session directory does not trigger a sweep")
}

func TestRunAtomicCacheLeavesNoTemporaryFiles(t *testing.T) {
	t.Parallel()

	dir, cache := t.TempDir(), t.TempDir()
	script(t, dir, "h.sh", "#!/bin/sh\nprintf x\n", 0o700)
	runOne(t, hook.Discover(dir), options(cache))

	entries, err := os.ReadDir(filepath.Join(cache, "s1"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "h.sh", entries[0].Name())
	data, err := os.ReadFile(filepath.Join(cache, "s1", "h.sh"))
	require.NoError(t, err)
	assert.Equal(t, "1760000000\nx\n", string(data))
}

func TestSafeSession(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "abc-1_Z", hook.SafeSession("abc-1_Z"))
	for _, bad := range []string{"", "..", "a/b", "a b", "é"} {
		assert.Equal(t, "shared", hook.SafeSession(bad), "%q", bad)
	}
}
