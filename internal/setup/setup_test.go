package setup_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

const testBin = "/opt/cs/bin/claude-statusline"

var prettySettings = "{\n  \"theme\": \"dark\",\n  \"env\": {\n    \"B\": \"1\",\n    \"A\": \"2\"\n  },\n  \"model\": \"opus\",\n  \"alpha\": [1, 2]\n}\n"

func newDir(t *testing.T, settings *string) (dir, path string) {
	t.Helper()
	dir = t.TempDir()
	path = filepath.Join(dir, "settings.json")
	if settings != nil {
		require.NoError(t, os.WriteFile(path, []byte(*settings), 0o640))
	}
	return dir, path
}

func locations(dir string) setup.Locations {
	return setup.Locations{ConfigDir: dir, StateDir: filepath.Join(dir, "claude-statusline"), Bin: testBin}
}

func run(dir, event string) string {
	return setup.Apply(locations(dir), event, "1.2.3")
}

func sentinelExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "claude-statusline", ".bootstrapped"))
	return err == nil
}

func TestApply(t *testing.T) {
	t.Parallel()

	own := `{"statusLine":{"type":"command","command":"` + testBin + `"},"subagentStatusLine":{"type":"command","command":"` + testBin + ` subagent"}}`
	other := `{"statusLine":{"type":"command","command":"/usr/bin/mine"},"subagentStatusLine":{"type":"command","command":"/usr/bin/mine"}}`
	otherSub := `{"statusLine":{"type":"command","command":"` + testBin + `"},"subagentStatusLine":{"type":"command","command":"/usr/bin/mine"}}`
	invalid := "not json"
	array := "[1]"

	tests := []struct {
		name         string
		settings     *string
		sentinel     bool
		event        string
		wantMsg      string
		wantSettings string
		wantSentinel bool
	}{
		{"does not re-add lines the user removed", &prettySettings, true, setup.EventCurrent, "", prettySettings, true},
		{"announces an update without touching settings", &prettySettings, true, setup.EventUpdated, "updated to v1.2.3", prettySettings, true},
		{"does not re-add lines the user removed, even on install", &prettySettings, true, setup.EventInstalled, "", prettySettings, true},
		{"leaves invalid JSON alone and says so on install", &invalid, false, setup.EventInstalled, "not valid JSON", invalid, false},
		{"leaves invalid JSON alone silently on later events", &invalid, false, setup.EventCurrent, "", invalid, false},
		{"leaves a non-object alone", &array, false, setup.EventInstalled, "not valid JSON", array, false},
		{"keeps our own statusLine and says installed", &own, false, setup.EventInstalled, "v1.2.3 installed.", own, true},
		{"keeps our own statusLine silently on current", &own, false, setup.EventCurrent, "", own, true},
		{"leaves someone else's statusLine and explains the switch", &other, false, setup.EventInstalled, "already have a statusLine", other, true},
		{"leaves someone else's statusLine silently on current", &other, false, setup.EventCurrent, "", other, true},
		{"leaves someone else's subagentStatusLine and explains the switch", &otherSub, false, setup.EventInstalled, "already have a subagentStatusLine", otherSub, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, path := newDir(t, tt.settings)
			if tt.sentinel {
				require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped")))
				require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped-subagent")))
			}

			msg := run(dir, tt.event)

			if tt.wantMsg == "" {
				assert.Empty(t, msg)
			} else {
				assert.Contains(t, msg, tt.wantMsg)
			}
			got, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSettings, string(got))
			assert.Equal(t, tt.wantSentinel, sentinelExists(dir))
			assert.NoFileExists(t, path+".bak")
		})
	}
}

func TestInsertStatusLine(t *testing.T) {
	t.Parallel()

	block := func(indent, nl string) string {
		return `"statusLine": {` + nl + indent + indent + `"type": "command",` + nl +
			indent + indent + `"command": "` + testBin + `",` + nl +
			indent + indent + `"refreshInterval": 60` + nl + indent + "}"
	}
	compact := `"statusLine":{"type":"command","command":"` + testBin + `","refreshInterval":60}`

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"two-space file",
			"{\n  \"theme\": \"dark\",\n  \"model\": \"opus\"\n}\n",
			"{\n  \"theme\": \"dark\",\n  \"model\": \"opus\",\n  " + block("  ", "\n") + "\n}\n",
		},
		{
			"four-space file",
			"{\n    \"a\": 1\n}\n",
			"{\n    \"a\": 1,\n    " + block("    ", "\n") + "\n}\n",
		},
		{
			"tab-indented file",
			"{\n\t\"a\": 1,\n\t\"b\": 2\n}\n",
			"{\n\t\"a\": 1,\n\t\"b\": 2,\n\t" + block("\t", "\n") + "\n}\n",
		},
		{
			"compact one-line file",
			`{"a":1,"b":"x"}` + "\n",
			`{"a":1,"b":"x",` + compact + "}\n",
		},
		{
			"empty object",
			"{}",
			"{\n  " + block("  ", "\n") + "\n}",
		},
		{
			"empty object with a newline",
			"{\n}\n",
			"{\n  " + block("  ", "\n") + "\n}\n",
		},
		{
			"empty object with a space",
			"{ }\n",
			"{\n  " + block("  ", "\n") + "\n}\n",
		},
		{
			"file without a trailing newline",
			"{\n  \"a\": 1\n}",
			"{\n  \"a\": 1,\n  " + block("  ", "\n") + "\n}",
		},
		{
			"last member is a nested object",
			"{\n  \"env\": {\n    \"A\": \"1\"\n  }\n}\n",
			"{\n  \"env\": {\n    \"A\": \"1\"\n  },\n  " + block("  ", "\n") + "\n}\n",
		},
		{
			"last member is an array",
			"{\n  \"a\": [1, 2]\n}\n",
			"{\n  \"a\": [1, 2],\n  " + block("  ", "\n") + "\n}\n",
		},
		{
			"closing braces inside strings",
			"{\n  \"a\": \"}\",\n  \"b\": \"x}\"\n}\n",
			"{\n  \"a\": \"}\",\n  \"b\": \"x}\",\n  " + block("  ", "\n") + "\n}\n",
		},
		{
			"CRLF line endings",
			"{\r\n  \"a\": 1\r\n}\r\n",
			"{\r\n  \"a\": 1,\r\n  " + block("  ", "\r\n") + "\r\n}\r\n",
		},
	}

	for _, tt := range tests {
		t.Run("inserts into a "+tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := setup.InsertMember([]byte(tt.in), "statusLine", []byte(`{"type":"command","command":"`+testBin+`","refreshInterval":60}`))

			require.True(t, ok, "insertion failed")
			assert.Equal(t, tt.want, string(got))
			assert.True(t, gjson.ValidBytes(got), "result is not valid JSON: %s", got)
		})
	}
}

func TestApplyWiring(t *testing.T) {
	t.Parallel()

	t.Run("writes the inserted entry, a backup of the original and the sentinel", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &prettySettings)

		msg := run(dir, setup.EventInstalled)

		assert.Contains(t, msg, "installed and enabled")
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, testBin, gjson.GetBytes(got, "statusLine.command").String())
		assert.EqualValues(t, 60, gjson.GetBytes(got, "statusLine.refreshInterval").Int())
		bak, err := os.ReadFile(path + ".bak")
		require.NoError(t, err)
		assert.Equal(t, prettySettings, string(bak))
		assert.True(t, sentinelExists(dir), "no sentinel")
	})
}

func TestApplySubagent(t *testing.T) {
	t.Parallel()

	subLine := `"subagentStatusLine": {` + "\n" + `    "type": "command",` + "\n" + `    "command": "` + testBin + ` subagent"` + "\n" + `  }`
	ownStatus := "{\n  \"statusLine\": {\"type\": \"command\", \"command\": \"" + testBin + "\"}\n}\n"
	mineStatus := "{\n  \"statusLine\": {\"type\": \"command\", \"command\": \"/usr/bin/mine\"}\n}\n"
	mineSub := "{\n  \"subagentStatusLine\": {\"type\": \"command\", \"command\": \"/usr/bin/mine\"}\n}\n"

	tests := []struct {
		name         string
		settings     string
		sentinel     bool
		event        string
		wantMsg      string
		wantStatus   string
		wantSub      string
		wantBackup   bool
		wantSentinel bool
	}{
		{"both absent wires both", prettySettings, false, setup.EventInstalled, "also styles the subagent rows", testBin, testBin + " subagent", true, true},
		{"our statusLine from an older install gains the subagent line once", ownStatus, true, setup.EventUpdated, "claude-statusline now also styles the subagent rows", testBin, testBin + " subagent", true, true},
		{"a user's own subagentStatusLine is kept", mineSub, false, setup.EventInstalled, "already have a subagentStatusLine", testBin, "/usr/bin/mine", true, true},
		{"a user's statusLine does not stop the subagent line", mineStatus, true, setup.EventCurrent, "claude-statusline now also styles the subagent rows", "/usr/bin/mine", testBin + " subagent", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, path := newDir(t, &tt.settings)
			if tt.sentinel {
				require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped")))
			}

			msg := run(dir, tt.event)

			assert.Contains(t, msg, tt.wantMsg)
			assert.Equal(t, 0, strings.Count(msg, "\n"), "message must be one line")
			got, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, gjson.GetBytes(got, "statusLine.command").String())
			assert.Equal(t, tt.wantSub, gjson.GetBytes(got, "subagentStatusLine.command").String())
			assert.True(t, gjson.ValidBytes(got))
			if tt.wantBackup {
				bak, err := os.ReadFile(path + ".bak")
				require.NoError(t, err)
				assert.Equal(t, tt.settings, string(bak))
			}
			assert.True(t, sentinelExists(dir))
			assert.FileExists(t, filepath.Join(dir, "claude-statusline", ".bootstrapped-subagent"))
		})
	}

	t.Run("adds the subagent line with the file's indentation", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &ownStatus)
		require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped")))

		run(dir, setup.EventCurrent)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSuffix(ownStatus, "\n}\n")+",\n  "+subLine+"\n}\n", string(got))
	})

	t.Run("is idempotent and writes a single backup of the original", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &prettySettings)

		first := run(dir, setup.EventInstalled)
		after, err := os.ReadFile(path)
		require.NoError(t, err)
		again := run(dir, setup.EventCurrent)
		afterAgain, err := os.ReadFile(path)
		require.NoError(t, err)

		assert.Contains(t, first, "installed and enabled")
		assert.Empty(t, again)
		assert.Equal(t, string(after), string(afterAgain))
		bak, err := os.ReadFile(path + ".bak")
		require.NoError(t, err)
		assert.Equal(t, prettySettings, string(bak))
		assert.Equal(t, testBin, gjson.GetBytes(after, "statusLine.command").String())
		assert.Equal(t, testBin+" subagent", gjson.GetBytes(after, "subagentStatusLine.command").String())
	})

	t.Run("quotes a binary path that contains a space", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &prettySettings)

		loc := locations(dir)
		loc.Bin = "/opt/my tools/claude-statusline"
		setup.Apply(loc, setup.EventInstalled, "1.2.3")

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "'/opt/my tools/claude-statusline' subagent", gjson.GetBytes(got, "subagentStatusLine.command").String())
	})
}

func TestApplyFiles(t *testing.T) {
	t.Parallel()

	t.Run("preserves the file mode", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &prettySettings)

		run(dir, setup.EventInstalled)

		fi, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o640), fi.Mode().Perm())
	})

	t.Run("creates a missing settings file with mode 0600", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, nil)

		msg := run(dir, setup.EventInstalled)

		assert.Contains(t, msg, "installed and enabled")
		fi, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(got), testBin)
	})

	t.Run("does nothing on a second run", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &prettySettings)
		run(dir, setup.EventInstalled)
		first, err := os.ReadFile(path)
		require.NoError(t, err)

		msg := run(dir, setup.EventCurrent)

		second, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.True(t, bytes.Equal(first, second), "second run changed the settings")
	})

	t.Run("leaves no temp files behind", func(t *testing.T) {
		t.Parallel()
		dir, _ := newDir(t, &prettySettings)
		run(dir, setup.EventInstalled)

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		for _, e := range entries {
			assert.False(t, strings.HasPrefix(e.Name(), ".settings-"), "left %s behind", e.Name())
		}
	})
}

func TestRun(t *testing.T) {
	t.Run("prints a single JSON line", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		var out bytes.Buffer
		setup.Run(&out, setup.EventInstalled, "1.2.3", locations(dir))

		var msg struct {
			SystemMessage string `json:"systemMessage"`
		}
		require.NoError(t, json.Unmarshal(out.Bytes(), &msg), "output %q", out.String())
		assert.Contains(t, msg.SystemMessage, "installed and enabled")
		assert.Equal(t, 1, strings.Count(out.String(), "\n"), "output = %q", out.String())
	})

	t.Run("prints nothing when there is nothing to say", func(t *testing.T) {
		t.Parallel()
		loc := locations(t.TempDir())
		var out bytes.Buffer
		setup.Run(&out, setup.EventInstalled, "1.2.3", loc)
		out.Reset()
		setup.Run(&out, setup.EventCurrent, "1.2.3", loc)
		assert.Zero(t, out.Len(), "output = %q", out.String())
	})

	t.Run("prints nothing without a config directory, state directory or binary", func(t *testing.T) {
		t.Parallel()
		var out bytes.Buffer
		setup.Run(&out, setup.EventInstalled, "1.2.3", setup.Locations{})
		setup.Uninstall(&out, setup.Locations{})
		assert.Zero(t, out.Len(), "output = %q", out.String())
	})
}

func settingsWith(status, sub string) string {
	members := []string{`"theme": "dark"`}
	if status != "" {
		members = append(members, `"statusLine": {"type": "command", "command": `+quote(status)+`, "padding": 0}`)
	}
	if sub != "" {
		members = append(members, `"subagentStatusLine": {"type": "command", "command": `+quote(sub)+`}`)
	}
	return "{\n  " + strings.Join(members, ",\n  ") + "\n}\n"
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestApplyOwnership(t *testing.T) {
	t.Parallel()

	const (
		pluginBin = "/old/claude/plugins/data/statusline-claude-statusline/bin/claude-statusline"
		shell     = "/Users/me/.claude/paddle-statusline.sh"
		subShell  = "bash /Users/me/.claude/paddle-subagent-statusline.sh"
	)

	tests := []struct {
		name       string
		status     string
		sub        string
		wantStatus string
		wantSub    string
		wantWrite  bool
		wantMsg    string
	}{
		{"the current binary is left alone", testBin, testBin + " subagent", testBin, testBin + " subagent", false, ""},
		{"a quoted current binary is left alone", "'" + testBin + "'", "'" + testBin + "' subagent", "'" + testBin + "'", "'" + testBin + "' subagent", false, ""},
		{"the legacy binary is repointed", "LEGACY", "LEGACY subagent", testBin, testBin + " subagent", true, "now provides your statusLine"},
		{"the legacy binary with a stray subagent argument is repointed", "LEGACY subagent", "LEGACY", testBin, testBin + " subagent", true, "now provides your statusLine"},
		{"an earlier plugin data binary is repointed", pluginBin, pluginBin + " subagent", testBin, testBin + " subagent", true, "now provides your statusLine"},
		{"the old Paddle shell lines are repointed", shell, subShell, testBin, testBin + " subagent", true, "now provides your statusLine"},
		{"a shell script run through bash is repointed", "bash " + shell, "", testBin, "", true, "now provides your statusLine"},
		{"someone else's lines are left alone", "/usr/bin/mine", "/usr/bin/mine subagent", "/usr/bin/mine", "/usr/bin/mine subagent", false, "already have a statusLine"},
		{"a script run through sh is repointed", "sh " + shell, "", testBin, "", true, "now provides your statusLine"},
		{"a script run through a path to zsh is repointed", "/bin/zsh " + shell, "", testBin, "", true, "now provides your statusLine"},
		{"a wrapper that is passed the script is someone else's", "/usr/bin/wrap " + shell, "/usr/bin/wrap " + subShell, "/usr/bin/wrap " + shell, "/usr/bin/wrap " + subShell, false, "already have a statusLine"},
		{"a shell with extra arguments is someone else's", "bash -x " + shell, "", "bash -x " + shell, "", false, "already have a statusLine"},
		{"a script followed by arguments is someone else's", shell + " --flag", "", shell + " --flag", "", false, "already have a statusLine"},
		{"an interpreter that is not a plain shell is someone else's", "python3 " + shell, "", "python3 " + shell, "", false, "already have a statusLine"},
		{"the other key's script is someone else's", "/x/other.sh", "/x/paddle-statusline.sh", "/x/other.sh", "/x/paddle-statusline.sh", false, "already have a statusLine"},
		{"our binary with other arguments is someone else's", testBin + " --flag", "", testBin + " --flag", "", false, "already have a statusLine"},
		{"a similarly named binary elsewhere is someone else's", "/usr/local/bin/claude-statusline", "", "/usr/local/bin/claude-statusline", "", false, "already have a statusLine"},
		{"a bin directory outside the plugin data layout is someone else's", "/c/plugins/cache/x/bin/claude-statusline", "", "/c/plugins/cache/x/bin/claude-statusline", "", false, "already have a statusLine"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			legacy := filepath.Join(dir, "claude-statusline", "bin", "claude-statusline")
			swap := func(v string) string { return strings.ReplaceAll(v, "LEGACY", legacy) }
			before := settingsWith(swap(tt.status), swap(tt.sub))
			path := filepath.Join(dir, "settings.json")
			require.NoError(t, os.WriteFile(path, []byte(before), 0o600))
			require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped")))
			require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped-subagent")))

			msg := run(dir, setup.EventInstalled)

			got, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, swap(tt.wantStatus), gjson.GetBytes(got, "statusLine.command").String())
			assert.Equal(t, swap(tt.wantSub), gjson.GetBytes(got, "subagentStatusLine.command").String())
			assert.Contains(t, msg, tt.wantMsg)
			if tt.wantWrite {
				bak, err := os.ReadFile(path + ".bak")
				require.NoError(t, err)
				assert.Equal(t, before, string(bak))
				assert.EqualValues(t, 0, gjson.GetBytes(got, "statusLine.padding").Int(), "other keys of the entry are kept")
				assert.Contains(t, msg, "backed up at")
				assert.NotEqual(t, before, string(got))
			} else {
				assert.Equal(t, before, string(got))
				assert.NoFileExists(t, path+".bak")
			}
			assert.True(t, gjson.ValidBytes(got))
		})
	}

	t.Run("a second run writes nothing and says nothing", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		require.NoError(t, os.WriteFile(path, []byte(settingsWith("/Users/me/.claude/paddle-statusline.sh", "")), 0o600))
		require.Contains(t, run(dir, setup.EventInstalled), "now provides your statusLine")
		first, err := os.ReadFile(path)
		require.NoError(t, err)
		require.NoError(t, os.Remove(path+".bak"))

		msg := run(dir, setup.EventCurrent)

		second, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Equal(t, string(first), string(second))
		assert.NoFileExists(t, path+".bak")
	})

	t.Run("repointing keeps the file's formatting", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		require.NoError(t, os.WriteFile(path, []byte(settingsWith("/Users/me/.claude/paddle-statusline.sh", "")), 0o600))
		require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped")))
		require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped-subagent")))

		run(dir, setup.EventUpdated)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, settingsWith(testBin, ""), string(got))
	})

	t.Run("the old shell script is not taken over on a current run", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		before := settingsWith("/Users/me/.claude/paddle-statusline.sh", "bash /Users/me/.claude/paddle-subagent-statusline.sh")
		require.NoError(t, os.WriteFile(path, []byte(before), 0o600))
		require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped")))
		require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped-subagent")))

		msg := run(dir, setup.EventCurrent)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Equal(t, before, string(got))
		assert.NoFileExists(t, path+".bak")
	})

	t.Run("an earlier binary is still repointed on a current run", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		legacy := filepath.Join(dir, "claude-statusline", "bin", "claude-statusline")
		require.NoError(t, os.WriteFile(path, []byte(settingsWith(legacy, "")), 0o600))
		require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped")))
		require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped-subagent")))

		run(dir, setup.EventCurrent)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, testBin, gjson.GetBytes(got, "statusLine.command").String())
	})

	t.Run("a sentinel from the old layout stops a removed line coming back", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		require.NoError(t, os.WriteFile(path, []byte(prettySettings), 0o600))
		legacy := filepath.Join(dir, "claude-statusline")
		require.NoError(t, setup.Touch(filepath.Join(legacy, ".bootstrapped")))
		require.NoError(t, setup.Touch(filepath.Join(legacy, ".bootstrapped-subagent")))
		loc := setup.Locations{ConfigDir: dir, StateDir: filepath.Join(dir, "plugins", "data", "x"), Bin: filepath.Join(dir, "plugins", "data", "x", "bin", "claude-statusline")}

		msg := setup.Apply(loc, setup.EventInstalled, "1.2.3")

		assert.Empty(t, msg)
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, prettySettings, string(got))
		assert.FileExists(t, filepath.Join(loc.StateDir, ".bootstrapped"))
	})

	t.Run("a missing settings file is not created when the lines were already added", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped")))
		require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped-subagent")))

		assert.Empty(t, run(dir, setup.EventCurrent))

		assert.NoFileExists(t, filepath.Join(dir, "settings.json"))
	})
}

func TestApplyMigration(t *testing.T) {
	t.Parallel()

	newLayout := func(t *testing.T) (loc setup.Locations, legacy, settings string) {
		t.Helper()
		dir := t.TempDir()
		legacy = filepath.Join(dir, "claude-statusline")
		data := filepath.Join(dir, "plugins", "data", "statusline-claude-statusline")
		loc = setup.Locations{ConfigDir: dir, StateDir: data, Bin: filepath.Join(data, "bin", "claude-statusline")}
		for _, f := range []string{"bin/claude-statusline", ".bootstrapped", ".bootstrapped-subagent", "last-check", ".no-attestation-notice", "cache/ratelimits", "cache/hooks/s1/x", "tmp.abc/x", "config.yaml"} {
			require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(legacy, f)), 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(legacy, f), []byte("x"), 0o600))
		}
		settings = filepath.Join(dir, "settings.json")
		legacyBin := filepath.Join(legacy, "bin", "claude-statusline")
		require.NoError(t, os.WriteFile(settings, []byte(settingsWith(legacyBin, legacyBin+" subagent")), 0o600))
		return loc, legacy, settings
	}

	t.Run("removes the old install and keeps config.yaml once the settings are repointed", func(t *testing.T) {
		t.Parallel()
		loc, legacy, settings := newLayout(t)

		msg := setup.Apply(loc, setup.EventInstalled, "1.2.3")

		assert.Contains(t, msg, "now provides your statusLine")
		got, err := os.ReadFile(settings)
		require.NoError(t, err)
		assert.Equal(t, loc.Bin, gjson.GetBytes(got, "statusLine.command").String())
		assert.Equal(t, loc.Bin+" subagent", gjson.GetBytes(got, "subagentStatusLine.command").String())
		for _, gone := range []string{"bin", ".bootstrapped", ".bootstrapped-subagent", "last-check", ".no-attestation-notice", "cache", "tmp.abc"} {
			assert.NoDirExists(t, filepath.Join(legacy, gone))
			assert.NoFileExists(t, filepath.Join(legacy, gone))
		}
		assert.FileExists(t, filepath.Join(legacy, "config.yaml"))
	})

	t.Run("removes the old directory when nothing else is in it", func(t *testing.T) {
		t.Parallel()
		loc, legacy, _ := newLayout(t)
		require.NoError(t, os.Remove(filepath.Join(legacy, "config.yaml")))

		setup.Apply(loc, setup.EventInstalled, "1.2.3")

		assert.NoDirExists(t, legacy)
	})

	t.Run("leaves the old install while a setting still points at it", func(t *testing.T) {
		t.Parallel()
		loc, legacy, settings := newLayout(t)
		legacyBin := filepath.Join(legacy, "bin", "claude-statusline")
		require.NoError(t, os.WriteFile(settings, []byte(settingsWith("/usr/bin/mine", "wrapper "+legacyBin+" --extra")), 0o600))

		setup.Apply(loc, setup.EventInstalled, "1.2.3")

		assert.FileExists(t, legacyBin)
	})

	t.Run("does nothing when the binary runs from the legacy layout", func(t *testing.T) {
		t.Parallel()
		loc, legacy, _ := newLayout(t)
		loc.StateDir = legacy
		loc.Bin = filepath.Join(legacy, "bin", "claude-statusline")

		setup.Apply(loc, setup.EventCurrent, "1.2.3")

		assert.FileExists(t, filepath.Join(legacy, "bin", "claude-statusline"))
		assert.FileExists(t, filepath.Join(legacy, "last-check"))
	})

	t.Run("leaves the old install when the settings are not valid JSON", func(t *testing.T) {
		t.Parallel()
		loc, legacy, settings := newLayout(t)
		require.NoError(t, os.WriteFile(settings, []byte("not json"), 0o600))

		setup.Apply(loc, setup.EventInstalled, "1.2.3")

		assert.FileExists(t, filepath.Join(legacy, "bin", "claude-statusline"))
	})
}

func TestApplyMigrationCommandForms(t *testing.T) {
	t.Parallel()

	layout := func(t *testing.T) (loc setup.Locations, legacyBin, settings string) {
		t.Helper()
		home := t.TempDir()
		dir := filepath.Join(home, ".claude")
		data := filepath.Join(dir, "plugins", "data", "statusline-claude-statusline")
		loc = setup.Locations{ConfigDir: dir, StateDir: data, Bin: filepath.Join(data, "bin", "claude-statusline"), Home: home}
		legacyBin = filepath.Join(dir, "claude-statusline", "bin", "claude-statusline")
		require.NoError(t, os.MkdirAll(filepath.Dir(legacyBin), 0o700))
		require.NoError(t, os.WriteFile(legacyBin, []byte("x"), 0o700))
		return loc, legacyBin, filepath.Join(dir, "settings.json")
	}

	for i, form := range []string{
		"~/.claude/claude-statusline/bin/claude-statusline",
		"$HOME/.claude/claude-statusline/bin/claude-statusline",
		"${HOME}/.claude/claude-statusline/bin/claude-statusline",
		"$CLAUDE_CONFIG_DIR/claude-statusline/bin/claude-statusline",
		"${CLAUDE_CONFIG_DIR}/claude-statusline/bin/claude-statusline",
	} {
		t.Run(fmt.Sprintf("repoints form %d", i), func(t *testing.T) {
			t.Parallel()
			loc, legacyBin, settings := layout(t)
			require.NoError(t, os.WriteFile(settings, []byte(settingsWith(form, form+" subagent")), 0o600))

			msg := setup.Apply(loc, setup.EventCurrent, "1.2.3")

			assert.Contains(t, msg, "now provides your statusLine")
			got, err := os.ReadFile(settings)
			require.NoError(t, err)
			assert.Equal(t, loc.Bin, gjson.GetBytes(got, "statusLine.command").String())
			assert.Equal(t, loc.Bin+" subagent", gjson.GetBytes(got, "subagentStatusLine.command").String())
			assert.NoFileExists(t, legacyBin)
		})
	}

	for i, form := range []string{
		"$SOMEVAR/claude-statusline/bin/claude-statusline",
		"~/.claude/claude-statusline/bin/claude-statusline",
	} {
		t.Run(fmt.Sprintf("keeps the old install for form %d", i), func(t *testing.T) {
			t.Parallel()
			loc, legacyBin, settings := layout(t)
			if strings.HasPrefix(form, "~") {
				loc.Home = ""
			}
			before := settingsWith(form, form+" subagent")
			require.NoError(t, os.WriteFile(settings, []byte(before), 0o600))

			setup.Apply(loc, setup.EventCurrent, "1.2.3")

			got, err := os.ReadFile(settings)
			require.NoError(t, err)
			assert.Equal(t, before, string(got))
			assert.FileExists(t, legacyBin)
		})
	}

	t.Run("keeps the old install while only one key is repointed", func(t *testing.T) {
		t.Parallel()
		loc, legacyBin, settings := layout(t)
		require.NoError(t, os.WriteFile(settings, []byte(settingsWith(legacyBin, "$SOMEVAR/claude-statusline/bin/claude-statusline subagent")), 0o600))

		setup.Apply(loc, setup.EventCurrent, "1.2.3")

		assert.FileExists(t, legacyBin)
	})
}

func TestApplyManual(t *testing.T) {
	t.Parallel()

	t.Run("adds absent lines despite the markers", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, &prettySettings)
		require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped")))
		require.NoError(t, setup.Touch(filepath.Join(dir, "claude-statusline", ".bootstrapped-subagent")))

		msg := run(dir, setup.EventManual)

		assert.Contains(t, msg, "installed and enabled")
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, locations(dir).Bin, gjson.GetBytes(got, "statusLine.command").String())
		assert.Equal(t, locations(dir).Bin+" subagent", gjson.GetBytes(got, "subagentStatusLine.command").String())
	})

	t.Run("adds the lines back after an uninstall", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, nil)
		require.NoError(t, os.WriteFile(path, []byte(settingsWith(testBin, testBin+" subagent")), 0o600))
		setup.Uninstall(&bytes.Buffer{}, locations(dir))
		require.Empty(t, run(dir, setup.EventCurrent))
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		require.False(t, gjson.GetBytes(got, "statusLine").Exists())

		run(dir, setup.EventManual)

		got, err = os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, testBin, gjson.GetBytes(got, "statusLine.command").String())
	})

	t.Run("takes over the old shell script", func(t *testing.T) {
		t.Parallel()
		dir, path := newDir(t, nil)
		require.NoError(t, os.WriteFile(path, []byte(settingsWith("/Users/me/.claude/paddle-statusline.sh", "")), 0o600))

		run(dir, setup.EventManual)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, testBin, gjson.GetBytes(got, "statusLine.command").String())
	})
}

func TestSymlinkedSettings(t *testing.T) {
	t.Parallel()

	link := func(t *testing.T, content string) (dir, link, target string) {
		t.Helper()
		dir = t.TempDir()
		dotfiles := filepath.Join(dir, "dotfiles")
		require.NoError(t, os.MkdirAll(dotfiles, 0o700))
		target = filepath.Join(dotfiles, "settings.json")
		require.NoError(t, os.WriteFile(target, []byte(content), 0o600))
		link = filepath.Join(dir, "settings.json")
		require.NoError(t, os.Symlink(target, link))
		return dir, link, target
	}
	requireSymlink := func(t *testing.T, path string) {
		t.Helper()
		fi, err := os.Lstat(path)
		require.NoError(t, err)
		assert.NotZero(t, fi.Mode()&os.ModeSymlink, "%s is no longer a symlink", path)
	}

	t.Run("setup writes through the link and backs up beside the target", func(t *testing.T) {
		t.Parallel()
		dir, link, target := link(t, prettySettings)

		run(dir, setup.EventInstalled)

		requireSymlink(t, link)
		got, err := os.ReadFile(target)
		require.NoError(t, err)
		assert.Equal(t, testBin, gjson.GetBytes(got, "statusLine.command").String())
		bak, err := os.ReadFile(target + ".bak")
		require.NoError(t, err)
		assert.Equal(t, prettySettings, string(bak))
		assert.NoFileExists(t, link+".bak")
	})

	t.Run("uninstall writes through the link and backs up beside the target", func(t *testing.T) {
		t.Parallel()
		before := settingsWith(testBin, testBin+" subagent")
		dir, link, target := link(t, before)

		setup.Uninstall(&bytes.Buffer{}, locations(dir))

		requireSymlink(t, link)
		got, err := os.ReadFile(target)
		require.NoError(t, err)
		assert.False(t, gjson.GetBytes(got, "statusLine").Exists())
		bak, err := os.ReadFile(target + ".bak")
		require.NoError(t, err)
		assert.Equal(t, before, string(bak))
		assert.NoFileExists(t, link+".bak")
	})
}

func TestUninstall(t *testing.T) {
	t.Parallel()

	data := func(dir string) setup.Locations {
		state := filepath.Join(dir, "plugins", "data", "statusline-claude-statusline")
		return setup.Locations{ConfigDir: dir, StateDir: state, Bin: filepath.Join(state, "bin", "claude-statusline")}
	}

	tests := []struct {
		name        string
		status      string
		sub         string
		wantStatus  string
		wantSub     string
		wantOut     []string
		wantWritten bool
	}{
		{"removes both lines that point at the current binary", "CUR", "CUR subagent", "", "", []string{"Removed statusLine and subagentStatusLine"}, true},
		{"removes the legacy and earlier plugin data lines", "LEGACY", "/old/plugins/data/x/bin/claude-statusline subagent", "", "", []string{"Removed statusLine and subagentStatusLine"}, true},
		{"keeps someone else's lines", "/usr/bin/mine", "/usr/bin/mine subagent", "/usr/bin/mine", "/usr/bin/mine subagent", []string{"statusLine is not set by claude-statusline", "subagentStatusLine is not set by claude-statusline"}, false},
		{"keeps the old Paddle shell lines", "/Users/me/.claude/paddle-statusline.sh", "/Users/me/.claude/paddle-subagent-statusline.sh", "/Users/me/.claude/paddle-statusline.sh", "/Users/me/.claude/paddle-subagent-statusline.sh", []string{"not set by claude-statusline"}, false},
		{"removes ours and keeps theirs", "CUR", "/usr/bin/mine", "", "/usr/bin/mine", []string{"Removed statusLine from", "subagentStatusLine is not set by claude-statusline"}, true},
		{"reports lines that are absent", "", "", "", "", []string{"statusLine is not set.", "subagentStatusLine is not set."}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			loc := data(dir)
			legacy := filepath.Join(dir, "claude-statusline", "bin", "claude-statusline")
			swap := func(v string) string {
				return strings.ReplaceAll(strings.ReplaceAll(v, "LEGACY", legacy), "CUR", loc.Bin)
			}
			before := settingsWith(swap(tt.status), swap(tt.sub))
			path := filepath.Join(dir, "settings.json")
			require.NoError(t, os.WriteFile(path, []byte(before), 0o600))
			keep := []string{filepath.Join(dir, "claude-statusline", "config.yaml"), filepath.Join(dir, "statusline.d", "10-x.sh"), filepath.Join(dir, "usage-latest.json")}
			for _, f := range keep {
				require.NoError(t, os.MkdirAll(filepath.Dir(f), 0o700))
				require.NoError(t, os.WriteFile(f, []byte("x"), 0o600))
			}
			require.NoError(t, setup.Touch(filepath.Join(loc.StateDir, ".bootstrapped")))

			var out bytes.Buffer
			setup.Uninstall(&out, loc)

			got, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, swap(tt.wantStatus), gjson.GetBytes(got, "statusLine.command").String())
			assert.Equal(t, swap(tt.wantSub), gjson.GetBytes(got, "subagentStatusLine.command").String())
			assert.Equal(t, "dark", gjson.GetBytes(got, "theme").String())
			assert.True(t, gjson.ValidBytes(got), "result is not valid JSON: %s", got)
			for _, want := range tt.wantOut {
				assert.Contains(t, out.String(), want)
			}
			assert.Contains(t, out.String(), "/plugin uninstall statusline@claude-statusline")
			if tt.wantWritten {
				bak, err := os.ReadFile(path + ".bak")
				require.NoError(t, err)
				assert.Equal(t, before, string(bak))
			} else {
				assert.Equal(t, before, string(got))
				assert.NoFileExists(t, path+".bak")
			}
			for _, f := range keep {
				assert.FileExists(t, f)
			}
			assert.FileExists(t, filepath.Join(loc.StateDir, ".bootstrapped"))
			assert.FileExists(t, filepath.Join(loc.StateDir, ".bootstrapped-subagent"))
			assert.Contains(t, out.String(), "will not be added back")
			assert.Contains(t, out.String(), "claude-statusline setup")
		})
	}

	t.Run("a later session does not add the lines back", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		loc := data(dir)
		path := filepath.Join(dir, "settings.json")
		require.NoError(t, os.WriteFile(path, []byte(settingsWith(loc.Bin, loc.Bin+" subagent")), 0o600))
		setup.Uninstall(&bytes.Buffer{}, loc)
		after, err := os.ReadFile(path)
		require.NoError(t, err)

		setup.Apply(loc, setup.EventCurrent, "1.2.3")
		setup.Apply(loc, setup.EventInstalled, "1.2.3")

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, string(after), string(got))
	})

	t.Run("copes with a missing settings file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		var out bytes.Buffer

		setup.Uninstall(&out, data(dir))

		assert.Contains(t, out.String(), "nothing to remove")
		assert.Contains(t, out.String(), "/plugin uninstall")
		assert.NoFileExists(t, filepath.Join(dir, "settings.json"))
	})

	t.Run("leaves invalid JSON alone", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))
		var out bytes.Buffer

		setup.Uninstall(&out, data(dir))

		assert.Contains(t, out.String(), "left alone")
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "not json", string(got))
	})
}

func TestRemoveMember(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		key  string
		want string
	}{
		{"first of several", "{\n  \"a\": 1,\n  \"b\": 2\n}\n", "a", "{\n  \"b\": 2\n}\n"},
		{"middle", "{\n  \"a\": 1,\n  \"b\": {\"x\": 1},\n  \"c\": 3\n}\n", "b", "{\n  \"a\": 1,\n  \"c\": 3\n}\n"},
		{"last", "{\n  \"a\": 1,\n  \"b\": [1, 2]\n}\n", "b", "{\n  \"a\": 1\n}\n"},
		{"only member", "{\n  \"a\": 1\n}\n", "a", "{}\n"},
		{"compact", `{"a":1,"b":"}","c":3}`, "b", `{"a":1,"c":3}`},
		{"CRLF", "{\r\n  \"a\": 1,\r\n  \"b\": 2\r\n}\r\n", "b", "{\r\n  \"a\": 1\r\n}\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := setup.RemoveMember([]byte(tt.in), tt.key)
			require.True(t, ok)
			assert.Equal(t, tt.want, string(got))
			assert.True(t, gjson.ValidBytes(got))
		})
	}

	t.Run("reports a missing key", func(t *testing.T) {
		t.Parallel()
		_, ok := setup.RemoveMember([]byte(`{"a":1}`), "z")
		assert.False(t, ok)
	})
}

func TestShellWords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"  ", nil},
		{"/a/b", []string{"/a/b"}},
		{"/a/b subagent", []string{"/a/b", "subagent"}},
		{"'/a b/c' subagent", []string{"/a b/c", "subagent"}},
		{`"/a b/c"  x`, []string{"/a b/c", "x"}},
		{`'/it'\''s/c' x`, []string{"/it's/c", "x"}},
		{`/a\ b/c`, []string{"/a b/c"}},
		{"bash\t/x.sh", []string{"bash", "/x.sh"}},
		{`''`, []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, setup.ShellWords(tt.in))
		})
	}
}
