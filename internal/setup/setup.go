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

type outcome int

const (
	skipped outcome = iota
	inserted
	ours
	other
)

func apply(c config) string {
	appDir := filepath.Join(c.dir, "claude-statusline")
	mainSentinel := filepath.Join(appDir, ".bootstrapped")
	subSentinel := filepath.Join(appDir, ".bootstrapped-subagent")
	wantMain, wantSub := !exists(mainSentinel), !exists(subSentinel)
	if !wantMain && !wantSub {
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
		if c.event == EventInstalled {
			return "claude-statusline v" + c.version + " installed, but " + settings + " is not valid JSON so it was left alone. It will retry next session."
		}
		return ""
	}

	subCommand := shellWord(c.bin) + " subagent"
	out := data
	var status, sub outcome
	if wantMain {
		status, out = wire(out, "statusLine", c.bin, `{"type":"command","command":%s,"refreshInterval":60}`)
	}
	if wantSub {
		sub, out = wire(out, "subagentStatusLine", subCommand, `{"type":"command","command":%s}`)
	}
	if (wantMain && status == skipped) || (wantSub && sub == skipped) || !gjson.ValidBytes(out) {
		return ""
	}

	if !bytes.Equal(out, data) {
		if err := writeAtomic(settings+".bak", data, mode); err != nil {
			return ""
		}
		if err := writeAtomic(settings, out, mode); err != nil {
			return ""
		}
	}
	for sentinel, want := range map[string]bool{mainSentinel: wantMain, subSentinel: wantSub} {
		if want {
			if err := touch(sentinel); err != nil {
				return ""
			}
		}
	}
	return message(c, settings, subCommand, status, sub)
}

// wire adds the key unless the file already has it, reporting which of the two happened.
func wire(data []byte, key, command, entryFormat string) (outcome, []byte) {
	if existing := gjson.GetBytes(data, key); existing.Exists() {
		if existing.Get("command").String() == command {
			return ours, data
		}
		return other, data
	}
	quoted, err := json.Marshal(command)
	if err != nil {
		return skipped, data
	}
	out, ok := insertMember(data, key, []byte(strings.Replace(entryFormat, "%s", string(quoted), 1)))
	if !ok {
		return skipped, data
	}
	return inserted, out
}

func message(c config, settings, subCommand string, status, sub outcome) string {
	lead := "claude-statusline v" + c.version
	var sentences []string
	switch status {
	case inserted:
		sentences = append(sentences, lead+" installed and enabled. It appears after your first message.")
	case ours:
		if c.event == EventInstalled {
			sentences = append(sentences, lead+" installed.")
		}
	case other:
		sentences = append(sentences, lead+" installed, but you already have a statusLine so it was left unchanged. To switch, set statusLine.command in "+settings+" to "+c.bin+".")
	case skipped:
		if c.event == EventUpdated {
			sentences = append(sentences, "claude-statusline updated to v"+c.version+".")
		}
	}

	switch sub {
	case inserted:
		if status == inserted {
			sentences = append(sentences, "It also styles the subagent rows.")
		} else {
			sentences = append(sentences, "claude-statusline now also styles the subagent rows.")
		}
	case other:
		note := "You already have a subagentStatusLine so it was left unchanged. To switch, set subagentStatusLine.command in " + settings + " to " + subCommand + "."
		if len(sentences) == 0 {
			note = lead + ": " + note
		}
		sentences = append(sentences, note)
	}

	var added []string
	if status == inserted {
		added = append(added, "statusLine")
	}
	if sub == inserted {
		added = append(added, "subagentStatusLine")
	}
	if len(added) > 0 {
		sentences = append(sentences, "Your previous settings are backed up at "+settings+".bak. To disable it, remove the "+strings.Join(added, " and ")+" block from settings.json.")
	}
	return strings.Join(sentences, " ")
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// shellWord quotes a path for the shell only when it has characters the shell would split or expand.
func shellWord(s string) string {
	if !strings.ContainsFunc(s, needsQuoting) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func needsQuoting(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', strings.ContainsRune("/._-+@:", r):
		return false
	}
	return true
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
