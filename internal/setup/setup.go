// Package setup wires the status line into Claude Code's settings.json once.
package setup

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/pretty"

	appconfig "github.com/PaddleHQ/claude-statusline/internal/config"
)

// Events describe what the bootstrap hook just did to the binary.
const (
	EventInstalled = "installed"
	EventUpdated   = "updated"
	EventCurrent   = "current"
)

type config struct {
	dir     string
	bin     string
	event   string
	version string
}

// Run never fails the session: it writes at most one systemMessage line and no errors.
func Run(w io.Writer, event, version string) {
	dir := appconfig.Dir(os.Getenv)
	if dir == "" {
		return
	}
	bin, err := os.Executable()
	if err != nil {
		return
	}
	if resolved, err := filepath.EvalSymlinks(bin); err == nil {
		bin = resolved
	}
	emit(w, apply(config{dir: dir, bin: bin, event: event, version: version}))
}

func emit(w io.Writer, msg string) {
	if msg == "" {
		return
	}
	b, err := json.Marshal(struct {
		SystemMessage string `json:"systemMessage"`
	}{msg})
	if err != nil {
		return
	}
	_, _ = w.Write(append(b, '\n'))
}

func apply(c config) string {
	installed := c.event == EventInstalled
	sentinel := filepath.Join(c.dir, "claude-statusline", ".bootstrapped")
	if _, err := os.Stat(sentinel); err == nil {
		if c.event == EventUpdated {
			return "claude-statusline updated to v" + c.version + "."
		}
		return ""
	}

	settings := filepath.Join(c.dir, "settings.json")
	data, err := os.ReadFile(settings)
	mode := os.FileMode(0o600)
	switch {
	case os.IsNotExist(err):
		data = []byte("{}\n")
		if err := writeAtomic(settings, data, mode); err != nil {
			return ""
		}
	case err != nil:
		return ""
	default:
		if fi, err := os.Stat(settings); err == nil {
			mode = fi.Mode().Perm()
		}
	}

	if !gjson.ValidBytes(data) || !gjson.ParseBytes(data).IsObject() {
		if installed {
			return "claude-statusline v" + c.version + " installed, but " + settings + " is not valid JSON so it was left alone. It will retry next session."
		}
		return ""
	}

	if sl := gjson.GetBytes(data, "statusLine"); sl.Exists() {
		if err := touch(sentinel); err != nil {
			return ""
		}
		if sl.Get("command").String() == c.bin {
			if installed {
				return "claude-statusline v" + c.version + " installed."
			}
			return ""
		}
		return "claude-statusline v" + c.version + " installed, but you already have a statusLine so it was left unchanged. To switch, set statusLine.command in " + settings + " to " + c.bin + "."
	}

	command, err := json.Marshal(c.bin)
	if err != nil {
		return ""
	}
	entry := []byte(`{"type":"command","command":` + string(command) + `,"refreshInterval":60}`)
	out, ok := insertStatusLine(data, entry)
	if !ok || !gjson.ValidBytes(out) {
		return ""
	}
	if err := writeAtomic(settings+".bak", data, mode); err != nil {
		return ""
	}
	if err := writeAtomic(settings, out, mode); err != nil {
		return ""
	}
	if err := touch(sentinel); err != nil {
		return ""
	}
	return "claude-statusline v" + c.version + " installed and enabled. It appears after your first message. Your previous settings are backed up at " + settings + ".bak. To disable it, remove the statusLine block from settings.json."
}

func touch(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, nil, 0o644)
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".settings-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// insertStatusLine adds the entry as the last member of the root object, matching
// the file's indentation and line endings and leaving every other byte as it was.
func insertStatusLine(data, entry []byte) ([]byte, bool) {
	open := len(data) - len(bytes.TrimLeft(data, " \t\r\n"))
	closing := bytes.LastIndexByte(data, '}')
	if open >= closing || data[open] != '{' {
		return nil, false
	}

	newline := "\n"
	if bytes.Contains(data, []byte("\r\n")) {
		newline = "\r\n"
	}

	firstKey, lastEnd := -1, -1
	gjson.ParseBytes(data).ForEach(func(key, value gjson.Result) bool {
		if firstKey < 0 {
			firstKey = key.Index
		}
		lastEnd = value.Index + len(value.Raw)
		return true
	})

	var at, resume int
	var text string
	switch {
	case firstKey < 0:
		at, resume = open+1, closing
		text = newline + "  " + `"statusLine": ` + prettyValue(entry, "  ", newline) + newline
	case bytes.IndexByte(data[open+1:firstKey], '\n') >= 0:
		span := data[open+1 : firstKey]
		indent := string(span[bytes.LastIndexByte(span, '\n')+1:])
		at, resume = lastEnd, lastEnd
		text = "," + newline + indent + `"statusLine": ` + prettyValue(entry, indent, newline)
	default:
		at, resume = lastEnd, lastEnd
		text = `,"statusLine":` + string(entry)
	}

	out := make([]byte, 0, len(data)+len(text))
	out = append(out, data[:at]...)
	out = append(out, text...)
	return append(out, data[resume:]...), true
}

func prettyValue(raw []byte, indent, newline string) string {
	out := pretty.PrettyOptions(raw, &pretty.Options{Indent: indent, Prefix: indent, Width: 1 << 20})
	s := strings.TrimSpace(string(out))
	s = strings.ReplaceAll(s, "\n", newline)
	return s
}
