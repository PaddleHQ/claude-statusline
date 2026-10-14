// Package hook runs the user's own segment scripts, with a timeout and a per-session cache.
package hook

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
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
	sessionNone  = "shared"

	execAttempts   = 3
	execRetryDelay = 20 * time.Millisecond
)

var (
	slotLine    = regexp.MustCompile(`^#\s*statusline-slot:\s*([a-z]+)\s*$`)
	ttlLine     = regexp.MustCompile(`^#\s*statusline-ttl:\s*(\d+)`)
	helpLine    = regexp.MustCompile(`^#\s*statusline-help:\s*(.*?)\s*$`)
	numPrefix   = regexp.MustCompile(`^\d+-`)
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
	// Help is the statusline-help description, with control characters removed; empty when the script has none.
	Help string
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

// readHeader applies the first slot, ttl and help directive found in the script's opening lines.
func readHeader(h *Hook) {
	f, err := os.Open(h.Path) //nolint:gosec // G304: reading the header of the user's own hook script is the feature
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	data, _ := io.ReadAll(io.LimitReader(f, headerBytes))

	var seenSlot, seenTTL, seenHelp bool
	for i, line := range strings.Split(string(data), "\n") {
		if i >= headerLines {
			break
		}
		line = strings.TrimSuffix(line, "\r")
		if !seenSlot {
			seenSlot = h.applySlot(line)
		}
		if !seenTTL {
			seenTTL = h.applyTTL(line)
		}
		if !seenHelp {
			seenHelp = h.applyHelp(line)
		}
	}
}

func (h *Hook) applySlot(line string) bool {
	m := slotLine.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	if s := Slot(m[1]); s == SlotUsage || s == SlotEnd {
		h.Slot = s
	}
	return true
}

func (h *Hook) applyTTL(line string) bool {
	m := ttlLine.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	if n, ok := config.ParseNumber(m[1]); ok {
		h.TTL = time.Duration(n) * time.Second
	}
	return true
}

func (h *Hook) applyHelp(line string) bool {
	m := helpLine.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	h.Help = strings.Map(func(r rune) rune {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			return -1
		}
		return r
	}, m[1])
	return true
}

// SafeSession returns id when it is safe to use as a directory name, else "shared".
func SafeSession(id string) string {
	if sessionSafe.MatchString(id) {
		return id
	}
	return sessionNone
}

// Run produces each hook's line, reusing cached results within their TTL and running the rest concurrently.
func (o *Options) Run(hooks []Hook) []Result {
	results := make([]Result, len(hooks))
	dir := o.sessionDir()

	var wg sync.WaitGroup
	for i, h := range hooks {
		wg.Go(func() {
			results[i] = Result{Hook: h, Text: sanitize(o.value(h, dir))}
		})
	}
	wg.Wait()
	return results
}

func (o *Options) sessionDir() string {
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
func (o *Options) value(h Hook, dir string) string {
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
	if out, ok := o.run(h, dir); ok {
		text = firstLine(out)
	}
	if cachePath != "" {
		writeCache(dir, cachePath, now, text)
	}
	return text
}

func (o *Options) run(h Hook, dir string) ([]byte, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), o.Timeout)
	defer cancel()

	in, out, cleanup, ok := o.stdio(dir)
	if !ok {
		return nil, false
	}
	defer cleanup()

	if err := o.exec(ctx, h, in, out); err != nil || ctx.Err() != nil {
		return nil, false
	}
	if _, err := out.Seek(0, io.SeekStart); err != nil {
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(out, outputBytes))
	return data, err == nil
}

// exec retries "text file busy": Linux refuses to run a script that any process, even a fork that is
// about to exec something else, still holds open for writing, such as an editor saving the hook.
func (o *Options) exec(ctx context.Context, h Hook, in, out *os.File) error {
	var err error
	for attempt := 1; attempt <= execAttempts; attempt++ {
		cmd := exec.CommandContext(ctx, h.Path) //nolint:gosec // G204: launching the user's own hook script is the feature
		cmd.Stdin = in
		cmd.Stdout = out
		isolate(cmd)
		err = cmd.Run()
		if err == nil {
			return nil
		}
		if !errors.Is(err, syscall.ETXTBSY) {
			return fmt.Errorf("run hook %s: %w", h.Path, err)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("run hook %s: %w", h.Path, ctx.Err())
		case <-time.After(time.Duration(attempt) * execRetryDelay):
		}
	}
	return fmt.Errorf("run hook %s after %d attempts: %w", h.Path, execAttempts, err)
}

// stdio gives a hook real files for stdin and stdout, since pipes can lose output that exec has not
// finished copying when the machine is busy. The returned cleanup closes and removes both.
func (o *Options) stdio(dir string) (in, out *os.File, cleanup func(), ok bool) {
	scratch, err := os.MkdirTemp(dir, ".hook-io.")
	if err != nil && dir != "" {
		scratch, err = os.MkdirTemp("", ".hook-io.")
	}
	if err != nil {
		return nil, nil, nil, false
	}
	cleanup = func() {
		if in != nil {
			_ = in.Close()
		}
		if out != nil {
			_ = out.Close()
		}
		_ = os.RemoveAll(scratch)
	}
	in, out, err = createFiles(scratch, o.Input)
	if err != nil {
		cleanup()
		return nil, nil, nil, false
	}
	return in, out, cleanup, true
}

func createFiles(dir string, input []byte) (in, out *os.File, err error) {
	in, err = os.Create(filepath.Join(dir, "in")) //nolint:gosec // G304: inside a directory this process just created
	if err != nil {
		return nil, nil, err //nolint:wrapcheck // the caller only needs to know it failed
	}
	if _, err = in.Write(input); err != nil {
		return in, nil, err //nolint:wrapcheck // the caller only needs to know it failed
	}
	if _, err = in.Seek(0, io.SeekStart); err != nil {
		return in, nil, err //nolint:wrapcheck // the caller only needs to know it failed
	}
	out, err = os.Create(filepath.Join(dir, "out")) //nolint:gosec // G304: inside a directory this process just created
	return in, out, err                             //nolint:wrapcheck // the caller only needs to know it failed
}

func firstLine(out []byte) string {
	line, _, _ := strings.Cut(string(out), "\n")
	return strings.TrimSuffix(line, "\r")
}

func readCache(path string) (ts int64, value string) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: the cache file sits in the user's own cache directory
	if err != nil {
		return 0, ""
	}
	head, rest, _ := strings.Cut(string(data), "\n")
	parsed, err := strconv.ParseInt(head, 10, 64)
	if err != nil || parsed < 0 {
		return 0, ""
	}
	value, _, _ = strings.Cut(rest, "\n")
	return parsed, value
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

// sanitize drops control characters from a hook's line, keeping only color sequences, so that
// a hook can color its output but cannot move the cursor or otherwise redraw the terminal.
func sanitize(s string) string {
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
