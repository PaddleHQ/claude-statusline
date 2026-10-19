// Package setup wires the status line into Claude Code's settings.json once.
package setup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	appconfig "github.com/PaddleHQ/claude-statusline/internal/config"

	"github.com/tidwall/gjson"
	"github.com/tidwall/pretty"
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

type outcome int

const (
	skipped outcome = iota
	inserted
	ours
	other
)

func apply(c config) string {
	sentinel := filepath.Join(c.dir, "claude-statusline", ".bootstrapped")
	if exists(sentinel) {
		return upToDateMessage(c)
	}

	settings := filepath.Join(c.dir, "settings.json")
	data, mode, ok := readSettings(settings)
	if !ok {
		return ""
	}
	if !gjson.ValidBytes(data) || !gjson.ParseBytes(data).IsObject() {
		return invalidMessage(c, settings)
	}

	status, out := wire(data, c.bin)
	if status == skipped || !gjson.ValidBytes(out) {
		return ""
	}
	if !bytes.Equal(out, data) && !backUpAndWrite(settings, data, out, mode) {
		return ""
	}
	if touch(sentinel) != nil {
		return ""
	}
	return message(c, settings, status)
}

func upToDateMessage(c config) string {
	if c.event == EventUpdated {
		return "claude-statusline updated to v" + c.version + "."
	}
	return ""
}

func invalidMessage(c config, settings string) string {
	if c.event != EventInstalled {
		return ""
	}
	return "claude-statusline v" + c.version + " installed, but " + settings +
		" is not valid JSON so it was left alone. It will retry next session."
}

// readSettings returns the settings file, creating an empty one when it is missing, and the mode to write it back with.
func readSettings(settings string) (data []byte, mode os.FileMode, ok bool) {
	data, err := os.ReadFile(settings) //nolint:gosec // G304: settings.json inside the Claude config directory is the file this command edits
	mode = 0o600
	switch {
	case os.IsNotExist(err):
		data = []byte("{}\n")
		if err := writeAtomic(settings, data, mode); err != nil {
			return nil, mode, false
		}
	case err != nil:
		return nil, mode, false
	default:
		if fi, err := os.Stat(settings); err == nil {
			mode = fi.Mode().Perm()
		}
	}
	return data, mode, true
}

func backUpAndWrite(settings string, before, after []byte, mode os.FileMode) bool {
	if err := writeAtomic(settings+".bak", before, mode); err != nil {
		return false
	}
	return writeAtomic(settings, after, mode) == nil
}

// wire adds the statusLine entry unless the file already has one, reporting which of the two happened.
func wire(data []byte, command string) (result outcome, out []byte) {
	if existing := gjson.GetBytes(data, "statusLine"); existing.Exists() {
		if existing.Get("command").String() == command {
			return ours, data
		}
		return other, data
	}
	quoted, err := json.Marshal(command)
	if err != nil {
		return skipped, data
	}
	out, ok := insertMember(data, "statusLine", []byte(`{"type":"command","command":`+string(quoted)+`,"refreshInterval":60}`))
	if !ok {
		return skipped, data
	}
	return inserted, out
}

func message(c config, settings string, status outcome) string {
	lead := "claude-statusline v" + c.version
	sentences := statusSentences(c, lead, settings, status)
	if status == inserted {
		sentences = append(sentences, "Your previous settings are backed up at "+settings+
			".bak. To disable it, remove the statusLine block from settings.json.")
	}
	return strings.Join(sentences, " ")
}

func statusSentences(c config, lead, settings string, status outcome) []string {
	switch status {
	case inserted:
		return []string{lead + " installed and enabled. It appears after your first message."}
	case ours:
		if c.event == EventInstalled {
			return []string{lead + " installed."}
		}
	case other:
		return []string{lead + " installed, but you already have a statusLine so it was left unchanged. To switch, set statusLine.command in " +
			settings + " to " + c.bin + "."}
	case skipped:
		if c.event == EventUpdated {
			return []string{"claude-statusline updated to v" + c.version + "."}
		}
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func touch(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create sentinel directory: %w", err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		return fmt.Errorf("write sentinel: %w", err)
	}
	return nil
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}
	f, err := os.CreateTemp(dir, ".settings-*")
	if err != nil {
		return fmt.Errorf("create temporary settings file: %w", err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("write temporary settings file: %w", err)
	}
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		return fmt.Errorf("set settings file mode: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temporary settings file: %w", err)
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("replace settings file: %w", err)
	}
	return nil
}

// insertMember adds key with the entry as the last member of the root object, matching
// the file's indentation and line endings and leaving every other byte as it was.
func insertMember(data []byte, key string, entry []byte) ([]byte, bool) {
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
		text = newline + "  " + `"` + key + `": ` + prettyValue(entry, "  ", newline) + newline
	case bytes.IndexByte(data[open+1:firstKey], '\n') >= 0:
		span := data[open+1 : firstKey]
		indent := string(span[bytes.LastIndexByte(span, '\n')+1:])
		at, resume = lastEnd, lastEnd
		text = "," + newline + indent + `"` + key + `": ` + prettyValue(entry, indent, newline)
	default:
		at, resume = lastEnd, lastEnd
		text = `,"` + key + `":` + string(entry)
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
