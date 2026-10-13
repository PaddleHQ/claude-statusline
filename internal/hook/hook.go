// Package hook runs the user's own segment scripts, with a timeout and a per-session cache.
package hook

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/PaddleHQ/claude-statusline/internal/config"
)

// Slot says where a hook's output goes when the segments list does not place it.
type Slot string

const (
	// SlotCtx puts the output right after the context segment.
	SlotCtx Slot = "ctx"
	// SlotUsage puts the output after the token and cache segments.
	SlotUsage Slot = "usage"
	// SlotEnd puts the output at the end of the line.
	SlotEnd Slot = "end"
)

const (
	headerLines = 20
	// headerBytes bounds how much of a script is read to find its directives.
	headerBytes = 64 * 1024
	// outputBytes bounds how much of a hook's output is kept.
	outputBytes  = 64 * 1024
	keepSessions = 7 * 24 * time.Hour
	// waitDelay bounds how long to wait for a killed hook's output pipe to close.
	waitDelay   = 200 * time.Millisecond
	sessionNone = "shared"
)

var (
	slotLine    = regexp.MustCompile(`^#\s*statusline-slot:\s*([a-z]+)\s*$`)
	ttlLine     = regexp.MustCompile(`^#\s*statusline-ttl:\s*([0-9]+)`)
	numPrefix   = regexp.MustCompile(`^[0-9]+-`)
	sgrSeq      = regexp.MustCompile("^\x1b\\[[0-9;]*m")
	sessionSafe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

// Hook is one executable script in the hook directory.
type Hook struct {
	Path string
	File string
	// Name is File without a leading "NN-" and without its extension.
	Name string
	Slot Slot
	// TTL is how long a result is reused; negative means the default applies.
	TTL time.Duration
}

// Result is a hook and the line it contributes; Text is empty when it has nothing to show.
type Result struct {
	Hook
	Text string
}

// Options controls a run.
type Options struct {
	// Input is the session JSON handed to every hook on stdin.
	Input []byte
	// CacheDir is the directory that holds one subdirectory per session; empty disables caching.
	CacheDir string
	// Session names the subdirectory of CacheDir; anything unsafe in a path means "shared".
	Session string
	Now     time.Time
	Timeout time.Duration
	// DefaultTTL applies to hooks without a statusline-ttl directive.
	DefaultTTL time.Duration
}

// Discover lists every regular executable file in dir, in filename order. A missing directory has no hooks.
func Discover(dir string) []Hook {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var hooks []Hook
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			continue
		}
		h := Hook{Path: path, File: e.Name(), Name: nameOf(e.Name()), Slot: SlotCtx, TTL: -1}
		readHeader(&h)
		hooks = append(hooks, h)
	}
	return hooks
}

func nameOf(file string) string {
	name := strings.TrimSuffix(file, filepath.Ext(file))
	return numPrefix.ReplaceAllString(name, "")
}

// readHeader applies the first slot and ttl directive found in the script's opening lines.
func readHeader(h *Hook) {
	f, err := os.Open(h.Path)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	data, _ := io.ReadAll(io.LimitReader(f, headerBytes))

	var seenSlot, seenTTL bool
	for i, line := range strings.Split(string(data), "\n") {
		if i >= headerLines {
			break
		}
		line = strings.TrimSuffix(line, "\r")
		if m := slotLine.FindStringSubmatch(line); m != nil && !seenSlot {
			seenSlot = true
			if s := Slot(m[1]); s == SlotUsage || s == SlotEnd {
				h.Slot = s
			}
		}
		if m := ttlLine.FindStringSubmatch(line); m != nil && !seenTTL {
			seenTTL = true
			if n, ok := config.ParseNumber(m[1]); ok {
				h.TTL = time.Duration(n) * time.Second
			}
		}
	}
}

// SafeSession returns id when it is safe to use as a directory name, else "shared".
func SafeSession(id string) string {
	if sessionSafe.MatchString(id) {
		return id
	}
	return sessionNone
}

// Run produces each hook's line, reusing cached results within their TTL and running the rest concurrently.
func Run(hooks []Hook, o Options) []Result {
	results := make([]Result, len(hooks))
	dir := o.sessionDir()

	var wg sync.WaitGroup
	for i, h := range hooks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = Result{Hook: h, Text: sanitise(o.value(h, dir))}
		}()
	}
	wg.Wait()
	return results
}

func (o Options) sessionDir() string {
	if o.CacheDir == "" {
		return ""
	}
	dir := filepath.Join(o.CacheDir, SafeSession(o.Session))
	if _, err := os.Stat(dir); err != nil {
		_ = os.MkdirAll(dir, 0o700)
		sweep(o.CacheDir, dir, o.Now)
	}
	return dir
}

// sweep removes the cache of sessions untouched for a week, so they do not pile up.
func sweep(root, keep string, now time.Time) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range entries {
		path := filepath.Join(root, e.Name())
		if path == keep || !e.IsDir() {
			continue
		}
		if info, err := e.Info(); err == nil && now.Sub(info.ModTime()) > keepSessions {
			_ = os.RemoveAll(path)
		}
	}
}

// value returns the hook's current line: the cached one within its TTL, else a fresh run.
// A failed or timed-out run keeps the previous line, and either way waits another TTL.
func (o Options) value(h Hook, dir string) string {
	ttl := h.TTL
	if ttl < 0 {
		ttl = o.DefaultTTL
	}
	var cached string
	var ts int64
	cachePath := ""
	if dir != "" {
		cachePath = filepath.Join(dir, h.File)
		ts, cached = readCache(cachePath)
	}
	now := o.Now.Unix()
	if cachePath != "" && now-ts < int64(ttl/time.Second) {
		return cached
	}

	text := cached
	if out, ok := o.run(h); ok {
		text = firstLine(out)
	}
	if cachePath != "" {
		writeCache(dir, cachePath, now, text)
	}
	return text
}

func (o Options) run(h Hook) ([]byte, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), o.Timeout)
	defer cancel()

	var out limitedBuffer
	cmd := exec.CommandContext(ctx, h.Path)
	cmd.Stdin = bytes.NewReader(o.Input)
	cmd.Stdout = &out
	cmd.WaitDelay = waitDelay
	isolate(cmd)
	if err := cmd.Run(); err != nil || ctx.Err() != nil {
		return nil, false
	}
	return out.buf.Bytes(), true
}

type limitedBuffer struct{ buf bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := outputBytes - b.buf.Len(); room > 0 {
		b.buf.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}

func firstLine(out []byte) string {
	line, _, _ := strings.Cut(string(out), "\n")
	return strings.TrimSuffix(line, "\r")
}

func readCache(path string) (int64, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, ""
	}
	head, rest, _ := strings.Cut(string(data), "\n")
	ts, err := strconv.ParseInt(head, 10, 64)
	if err != nil || ts < 0 {
		return 0, ""
	}
	value, _, _ := strings.Cut(rest, "\n")
	return ts, value
}

// writeCache replaces the file atomically so a concurrent render never reads half a write.
func writeCache(dir, path string, ts int64, value string) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, ".hook.")
	if err != nil {
		return
	}
	_, werr := tmp.WriteString(strconv.FormatInt(ts, 10) + "\n" + value + "\n")
	cerr := tmp.Close()
	if werr != nil || cerr != nil || os.Rename(tmp.Name(), path) != nil {
		_ = os.Remove(tmp.Name())
	}
}

// sanitise drops control characters from a hook's line, keeping only colour sequences, so that
// a hook can colour its output but cannot move the cursor or otherwise redraw the terminal.
func sanitise(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if m := sgrSeq.FindString(s[i:]); m != "" {
			b.WriteString(m)
			i += len(m)
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			continue
		}
		b.WriteString(s[i-size : i])
	}
	if strings.TrimSpace(b.String()) == "" {
		return ""
	}
	return b.String()
}
