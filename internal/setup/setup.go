// Package setup wires the status line into Claude Code's settings.json, and takes it out again.
package setup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	appconfig "github.com/PaddleHQ/claude-statusline/internal/config"

	"github.com/tidwall/gjson"
	"github.com/tidwall/pretty"
)

// Events describe what the bootstrap hook just did to the binary.
const (
	EventInstalled = "installed"
	EventUpdated   = "updated"
	EventCurrent   = "current"
	// EventManual is a setup run by the user, which adds absent lines even when an earlier run or uninstall left a marker.
	EventManual = "manual"
)

const (
	keyStatus = "statusLine"
	keySub    = "subagentStatusLine"

	binName          = "claude-statusline"
	subagentArg      = "subagent"
	statusScript     = "paddle-statusline.sh"
	subagentScript   = "paddle-subagent-statusline.sh"
	mainSentinelName = ".bootstrapped"
	subSentinelName  = ".bootstrapped-subagent"
)

// Locations says where setup reads and writes.
type Locations struct {
	// ConfigDir is the Claude config directory, which holds settings.json.
	ConfigDir string
	// StateDir is where the binary keeps its private state, including the markers that stop setup adding a line
	// the user removed.
	StateDir string
	// Bin is the running binary, with symlinks resolved.
	Bin string
	// Home is the user's home directory, which a leading ~ or $HOME in a command stands for.
	Home string
}

type config struct {
	Locations
	event   string
	version string
}

// Run never fails the session: it writes at most one systemMessage line and no errors.
func Run(w io.Writer, event, version string, loc Locations) {
	if loc.ConfigDir == "" || loc.StateDir == "" || loc.Bin == "" {
		return
	}
	emit(w, apply(config{Locations: loc, event: event, version: version}))
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
	replaced
	ours
	other
	declined
)

type claim int

const (
	foreign claim = iota
	current
	ownedBinary
	ownedScript
)

func (c *config) legacyDir() string {
	return filepath.Join(c.ConfigDir, binName)
}

func (c *config) legacyBins() []string {
	legacy := filepath.Join(c.legacyDir(), "bin", binName)
	bins := []string{legacy}
	if resolved, err := filepath.EvalSymlinks(legacy); err == nil && resolved != legacy {
		bins = append(bins, resolved)
	}
	return bins
}

func (c *config) subCommand() string {
	return shellWord(c.Bin) + " " + subagentArg
}

// classify says whether a status line command is the current binary, an earlier install of this tool, the old
// Paddle shell script, or something that belongs to someone else.
func (c *config) classify(key, command string) claim {
	words := shellWords(command)
	if len(words) == 0 {
		return foreign
	}
	wantArgs := []string(nil)
	script := statusScript
	if key == keySub {
		wantArgs = []string{subagentArg}
		script = subagentScript
	}
	path, args := filepath.Clean(c.expand(words[0])), words[1:]
	if len(args) == 0 || slices.Equal(args, []string{subagentArg}) {
		switch {
		case path == filepath.Clean(c.Bin) && slices.Equal(args, wantArgs):
			return current
		case path == filepath.Clean(c.Bin), slices.Contains(c.legacyBins(), path), isPluginDataBin(path):
			return ownedBinary
		}
	}
	if runsScript(words, script) {
		return ownedScript
	}
	return foreign
}

func (c *config) expand(word string) string {
	for _, p := range []struct{ prefix, value string }{
		{"~", c.Home},
		{"$HOME", c.Home},
		{"${HOME}", c.Home},
		{"$CLAUDE_CONFIG_DIR", c.ConfigDir},
		{"${CLAUDE_CONFIG_DIR}", c.ConfigDir},
	} {
		if rest, ok := strings.CutPrefix(word, p.prefix+"/"); ok && p.value != "" {
			return filepath.Join(p.value, rest)
		}
	}
	return word
}

// runsScript reports whether the command is the script itself, alone or as the only argument of a plain shell.
func runsScript(words []string, script string) bool {
	switch len(words) {
	case 1:
		return filepath.Base(words[0]) == script
	case 2:
		return slices.Contains([]string{"sh", "bash", "zsh", "dash"}, filepath.Base(words[0])) && filepath.Base(words[1]) == script
	}
	return false
}

func isBinaryClaim(cl claim) bool {
	return cl == current || cl == ownedBinary
}

func isPluginDataBin(path string) bool {
	return filepath.Base(path) == binName && appconfig.PluginDataDir(path) != ""
}

func (c *config) sentinel(name string) string {
	return filepath.Join(c.StateDir, name)
}

// wanted reports whether the line may be added when it is absent: not if an earlier run already did, since the user
// may have removed it since.
func (c *config) wanted(name string) bool {
	if c.event == EventManual {
		return true
	}
	if exists(c.sentinel(name)) {
		return false
	}
	return c.StateDir == c.legacyDir() || !exists(filepath.Join(c.legacyDir(), name))
}

func apply(c config) string {
	settings := filepath.Join(c.ConfigDir, "settings.json")
	target := resolveSymlinks(settings)
	wantMain, wantSub := c.wanted(mainSentinelName), c.wanted(subSentinelName)
	data, mode, ok := readSettings(target, wantMain || wantSub)
	if !ok {
		return ""
	}
	if !gjson.ValidBytes(data) || !gjson.ParseBytes(data).IsObject() {
		return invalidMessage(c, settings)
	}

	out, status, sub, ok := c.wireBoth(data, wantMain, wantSub)
	if !ok {
		return ""
	}
	if !bytes.Equal(out, data) && !backUpAndWrite(target, data, out, mode) {
		return ""
	}
	if !c.touchSentinels() {
		return ""
	}
	c.migrate(out)
	return message(c, settings, status, sub)
}

func invalidMessage(c config, settings string) string {
	if c.event != EventInstalled && c.event != EventManual {
		return ""
	}
	return fmt.Sprintf("claude-statusline v%s installed, but %s is not valid JSON so it was left alone. It will retry next session.",
		c.version, settings)
}

func (c *config) wireBoth(data []byte, wantMain, wantSub bool) (out []byte, status, sub outcome, ok bool) {
	out = data
	status, out = c.wire(out, keyStatus, c.Bin, `{"type":"command","command":%s,"refreshInterval":60}`, wantMain)
	sub, out = c.wire(out, keySub, c.subCommand(), `{"type":"command","command":%s}`, wantSub)
	if status == skipped || sub == skipped || !gjson.ValidBytes(out) {
		return nil, status, sub, false
	}
	return out, status, sub, true
}

// readSettings returns the settings file, creating an empty one when it is missing and create is set, and the mode to
// write it back with. A missing file that is not to be created reads as an empty object.
func readSettings(settings string, create bool) (data []byte, mode os.FileMode, ok bool) {
	data, err := os.ReadFile(settings) //nolint:gosec // G304: settings.json inside the Claude config directory is the file this command edits
	mode = 0o600
	switch {
	case os.IsNotExist(err):
		data = []byte("{}\n")
		if create {
			if err := writeAtomic(settings, data, mode); err != nil {
				return nil, mode, false
			}
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

func resolveSymlinks(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return path
}

func backUpAndWrite(settings string, before, after []byte, mode os.FileMode) bool {
	if err := writeAtomic(settings+".bak", before, mode); err != nil {
		return false
	}
	return writeAtomic(settings, after, mode) == nil
}

func (c *config) touchSentinels() bool {
	for _, name := range []string{mainSentinelName, subSentinelName} {
		if path := c.sentinel(name); !exists(path) && touch(path) != nil {
			return false
		}
	}
	return true
}

// wire sets the key to command when it is absent and allowed, or when its command is one this tool owns, and leaves a
// key that belongs to someone else alone.
func (c *config) wire(data []byte, key, command, entryFormat string, wantNew bool) (result outcome, out []byte) {
	existing := gjson.GetBytes(data, key)
	if !existing.Exists() {
		return insertEntry(data, key, command, entryFormat, wantNew)
	}

	cmd := existing.Get("command")
	switch cl := c.classify(key, cmd.String()); cl {
	case current:
		return ours, data
	case ownedBinary, ownedScript:
		if cl == ownedScript && !c.mayTakeOverScript() {
			return other, data
		}
		out, ok := replaceString(data, cmd, command)
		if !ok {
			return skipped, data
		}
		return replaced, out
	default:
		return other, data
	}
}

func insertEntry(data []byte, key, command, entryFormat string, wantNew bool) (result outcome, out []byte) {
	if !wantNew {
		return declined, data
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

func (c *config) mayTakeOverScript() bool {
	return c.event == EventInstalled || c.event == EventUpdated || c.event == EventManual
}

func replaceString(data []byte, value gjson.Result, text string) ([]byte, bool) {
	quoted, err := json.Marshal(text)
	if err != nil || value.Type != gjson.String || value.Index <= 0 || value.Index+len(value.Raw) > len(data) ||
		string(data[value.Index:value.Index+len(value.Raw)]) != value.Raw {
		return nil, false
	}
	out := make([]byte, 0, len(data)+len(quoted))
	out = append(out, data[:value.Index]...)
	out = append(out, quoted...)
	return append(out, data[value.Index+len(value.Raw):]...), true
}

// migrate removes what the old install layout left in the config directory once the settings no longer point at it.
// The user's config.yaml stays.
func (c *config) migrate(settings []byte) {
	legacy := c.legacyDir()
	if c.StateDir == legacy || !exists(legacy) || c.pointsIntoLegacy(settings) {
		return
	}
	for _, bin := range c.legacyBins() {
		if bytes.Contains(settings, []byte(bin)) {
			return
		}
	}
	for _, name := range []string{"bin", mainSentinelName, subSentinelName, "last-check", ".no-attestation-notice", "cache"} {
		_ = os.RemoveAll(filepath.Join(legacy, name))
	}
	if temps, err := filepath.Glob(filepath.Join(legacy, "tmp.*")); err == nil {
		for _, t := range temps {
			_ = os.RemoveAll(t)
		}
	}
	_ = os.Remove(legacy)
}

// pointsIntoLegacy reports whether a status line command mentions an install's bin directory in any form, other than
// the current binary.
func (c *config) pointsIntoLegacy(settings []byte) bool {
	for _, key := range []string{keyStatus, keySub} {
		cmd := gjson.GetBytes(settings, key+".command").String()
		if strings.Contains(cmd, binName+"/bin/") && c.classify(key, cmd) != current {
			return true
		}
	}
	return false
}

func message(c config, settings string, status, sub outcome) string {
	lead := "claude-statusline v" + c.version
	sentences := statusSentences(c, lead, settings, status)
	sentences = subSentences(c, sentences, lead, settings, status, sub)
	if len(sentences) == 0 && c.event == EventUpdated {
		return "claude-statusline updated to v" + c.version + "."
	}

	var changed []string
	if status == inserted || status == replaced {
		changed = append(changed, keyStatus)
	}
	if sub == inserted || sub == replaced {
		changed = append(changed, keySub)
	}
	if len(changed) > 0 {
		sentences = append(sentences, fmt.Sprintf("Your previous settings are backed up at %s.bak. To remove %s, run /statusline:uninstall.",
			settings, strings.Join(changed, " and ")))
	}
	return strings.Join(sentences, " ")
}

func statusSentences(c config, lead, settings string, status outcome) []string {
	switch status {
	case inserted:
		return []string{lead + " installed and enabled. It appears after your first message."}
	case replaced:
		return []string{lead + " now provides your statusLine. It appears after your first message."}
	case ours:
		if c.event == EventInstalled {
			return []string{lead + " installed."}
		}
	case other:
		if c.event != EventCurrent {
			return []string{fmt.Sprintf("%s installed, but you already have a statusLine so it was left unchanged. "+
				"To switch, set statusLine.command in %s to %s.", lead, settings, c.Bin)}
		}
	case skipped, declined:
	}
	return nil
}

func subSentences(c config, sentences []string, lead, settings string, status, sub outcome) []string {
	switch sub {
	case inserted, replaced:
		if status == inserted || status == replaced {
			return append(sentences, "It also styles the subagent rows.")
		}
		return append(sentences, "claude-statusline now also styles the subagent rows.")
	case other:
		if c.event == EventCurrent {
			return sentences
		}
		note := fmt.Sprintf("You already have a subagentStatusLine so it was left unchanged. "+
			"To switch, set subagentStatusLine.command in %s to %s.", settings, c.subCommand())
		if len(sentences) == 0 {
			note = lead + ": " + note
		}
		return append(sentences, note)
	default:
		return sentences
	}
}

// Uninstall removes the status line keys this tool owns from settings.json, with a backup, and says what it did.
func Uninstall(w io.Writer, loc Locations) {
	if loc.ConfigDir == "" || loc.Bin == "" {
		return
	}
	c := config{Locations: loc}
	settings := filepath.Join(c.ConfigDir, "settings.json")
	target := resolveSymlinks(settings)
	data, err := os.ReadFile(target) //nolint:gosec // G304: settings.json inside the Claude config directory is the file this command edits
	var lines []string
	switch {
	case os.IsNotExist(err):
		lines = append(lines, fmt.Sprintf("There is no %s, so there was nothing to remove.", settings))
	case err != nil || !gjson.ValidBytes(data) || !gjson.ParseBytes(data).IsObject():
		lines = append(lines, fmt.Sprintf("%s could not be read as a JSON object, so it was left alone.", settings))
	default:
		lines = append(lines, c.uninstallSettings(settings, target, data)...)
	}
	if c.StateDir != "" {
		c.touchSentinels()
	}
	lines = append(lines,
		"The lines will not be added back. To turn the status line on again, run claude-statusline setup.",
		"Your config.yaml, your statusline.d hooks and usage-latest.json were left in place.",
		"Next, run /plugin uninstall statusline@claude-statusline to remove the plugin and the binary it installed.")
	_, _ = fmt.Fprintln(w, strings.Join(lines, "\n"))
}

func (c *config) uninstallSettings(settings, target string, data []byte) []string {
	var lines, removed []string
	out := data
	for _, key := range []string{keyStatus, keySub} {
		existing := gjson.GetBytes(out, key)
		switch {
		case !existing.Exists():
			lines = append(lines, fmt.Sprintf("%s is not set.", key))
		case isBinaryClaim(c.classify(key, existing.Get("command").String())):
			next, ok := removeMember(out, key)
			if !ok {
				lines = append(lines, fmt.Sprintf("%s could not be removed, so it was left alone.", key))
				continue
			}
			out = next
			removed = append(removed, key)
		default:
			lines = append(lines, fmt.Sprintf("%s is not set by claude-statusline, so it was left alone.", key))
		}
	}
	if len(removed) > 0 {
		mode := os.FileMode(0o600)
		if fi, err := os.Stat(target); err == nil {
			mode = fi.Mode().Perm()
		}
		if gjson.ValidBytes(out) && backUpAndWrite(target, data, out, mode) {
			lines = append([]string{fmt.Sprintf("Removed %s from %s. The previous file is backed up at %s.bak.",
				strings.Join(removed, " and "), settings, settings)}, lines...)
		} else {
			lines = append([]string{fmt.Sprintf("Could not update %s, so it was left alone.", settings)}, lines...)
		}
	}
	return lines
}

// shellWords splits a command the way the shell would for the quoting this tool writes: whitespace separates words,
// and single quotes, double quotes and backslashes keep what they enclose together.
func shellWords(s string) []string {
	var sp splitter
	for _, r := range s {
		sp.add(r)
	}
	return sp.finish()
}

type splitter struct {
	words   []string
	cur     strings.Builder
	quote   rune
	inWord  bool
	escaped bool
}

func (sp *splitter) add(r rune) {
	switch {
	case sp.escaped:
		sp.cur.WriteRune(r)
		sp.escaped = false
	case sp.quote != 0:
		sp.addQuoted(r)
	case r == '\\':
		sp.escaped, sp.inWord = true, true
	case r == '\'' || r == '"':
		sp.quote, sp.inWord = r, true
	case unicode.IsSpace(r):
		sp.flush()
	default:
		sp.cur.WriteRune(r)
		sp.inWord = true
	}
}

func (sp *splitter) addQuoted(r rune) {
	if r == sp.quote {
		sp.quote = 0
		return
	}
	sp.cur.WriteRune(r)
}

func (sp *splitter) flush() {
	if sp.inWord {
		sp.words = append(sp.words, sp.cur.String())
		sp.cur.Reset()
		sp.inWord = false
	}
}

func (sp *splitter) finish() []string {
	sp.flush()
	return sp.words
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

// removeMember deletes the root member with the key, along with the comma that separated it from its neighbor,
// leaving every other byte as it was.
func removeMember(data []byte, key string) ([]byte, bool) {
	type span struct{ start, end int }
	var members []span
	target := -1
	gjson.ParseBytes(data).ForEach(func(k, v gjson.Result) bool {
		if k.String() == key {
			target = len(members)
		}
		members = append(members, span{k.Index, v.Index + len(v.Raw)})
		return true
	})
	if target < 0 {
		return nil, false
	}
	m := members[target]
	var from, to int
	switch {
	case target+1 < len(members):
		from, to = m.start, members[target+1].start
	case target > 0:
		from, to = members[target-1].end, m.end
	default:
		open := len(data) - len(bytes.TrimLeft(data, " \t\r\n"))
		from, to = open+1, bytes.LastIndexByte(data, '}')
	}
	if from < 0 || to < from || to > len(data) {
		return nil, false
	}
	out := make([]byte, 0, len(data))
	out = append(out, data[:from]...)
	return append(out, data[to:]...), true
}
