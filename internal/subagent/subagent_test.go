package subagent_test

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/subagent"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scenario is one run of the subagent command in a private home directory. The expected rows are the task id, \t for
// the tab, then the row content with an escape byte written as \e.
type scenario struct {
	name    string
	session string
	config  string
	env     map[string]string
	want    string
}

// rows runs the command and returns its rows in the form the expectations are written in.
func (s *scenario) rows(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	vars := map[string]string{"HOME": dir, "CLAUDE_CONFIG_DIR": dir}
	maps.Copy(vars, s.env)
	if s.config != "" {
		path := filepath.Join(dir, "claude-statusline", "config.yaml")
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(s.config+"\n"), 0o600))
	}

	out := subagent.Build([]byte(s.session), func(k string) string { return vars[k] })

	var rows []string
	for line := range strings.SplitSeq(strings.TrimSuffix(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		var row struct {
			ID      json.RawMessage `json:"id"`
			Content string          `json:"content"`
		}
		require.NoError(t, json.Unmarshal([]byte(line), &row), line)
		assert.NotRegexp(t, "[\x00-\x09\x0b-\x1a\x1c-\x1f\x7f]", row.Content, "content contains a control character other than newline and escape")
		rows = append(rows, string(row.ID)+`\t`+strings.ReplaceAll(row.Content, "\x1b", `\e`))
	}
	assert.NoFileExists(t, filepath.Join(dir, "usage-latest.json"), "the subagent command must not write the usage snapshot")
	return strings.Join(rows, "\n")
}

// TestBuild pins the row of each task: its name, model, effort, context bar, token count and description, and how the
// settings, the terminal width and NO_COLOR change them.
func TestBuild(t *testing.T) {
	t.Parallel()

	tests := []scenario{
		{
			name:    `Control characters and escape sequences in names, descriptions, models and efforts are stripped from the rows`,
			session: `{"tasks":[{"id":"t1","name":"bad\u001b[31mname","type":"local_agent","description":"line one\nline two \u001b]8;;http://evil\u0007link\u009b31m end\u0000","model":"claude-sonnet-4-5\u001b[0m","effort":"hi\u001bgh","contextWindowSize":200000,"tokenCount":50000}]}`,
			want:    `"t1"\t\e[1mbad[31mname\e[0m \e[2m·\e[0m sonnet-4-5[0m\e[2m [high]\e[0m \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░░░░░\e[0m \e[32m25%\e[0m \e[2m·\e[0m 50.0k \e[2m·\e[0m \e[2mline oneline two ]8;;http://evillink31m end\e[0m`,
		},
		{
			name:    `The ctx.width, ctx.warn, ctx.bad and separator settings apply to agent rows`,
			session: `{"tasks":[{"id":"t1","name":"reviewer","description":"Custom thresholds","model":"claude-sonnet-4-5","contextWindowSize":200000,"tokenCount":60000}]}`,
			config: `ctx:
  width: 5
  warn: 20
  bad: 25
separator: "|"`,
			want: `"t1"\t\e[1mreviewer\e[0m \e[2m|\e[0m sonnet-4-5 \e[2m|\e[0m ctx: \e[31m█▌\e[0m\e[2m░░░\e[0m \e[31m30%\e[0m \e[2m|\e[0m 60.0k \e[2m|\e[0m \e[2mCustom thresholds\e[0m`,
		},
		{
			name:    `Empty input prints no rows`,
			session: ``,
			want:    ``,
		},
		{
			name:    `An empty task list prints no rows`,
			session: `{"tasks":[]}`,
			want:    ``,
		},
		{
			name:    `The gradient theme's context bars in a 100 column panel, at each color threshold`,
			session: `{"columns":100,"tasks":[{"id":"ok","name":"ok","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":99999},{"id":"warn","name":"warn","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":100000},{"id":"bad","name":"bad","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":180000},{"id":"full","name":"full","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":200000},{"id":"over","name":"over","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":250000},{"id":"zero","name":"zero","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":0}]}`,
			config:  `theme: gradient`,
			want: `"ok"\t\e[1mok\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;188;226;173m▰\e[0m\e[38;2;228;226;174m▰\e[0m\e[38;2;74;74;74m▱▱▱\e[0m \e[32m49%\e[0m \e[2m·\e[0m 99.9k
"warn"\t\e[1mwarn\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;188;226;173m▰\e[0m\e[38;2;228;226;174m▰\e[0m\e[38;2;74;74;74m▱▱▱\e[0m \e[33m50%\e[0m \e[2m·\e[0m 100.0k
"bad"\t\e[1mbad\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;188;226;173m▰\e[0m\e[38;2;228;226;174m▰\e[0m\e[38;2;248;209;174m▰\e[0m\e[38;2;246;174;171m▰\e[0m\e[38;2;74;74;74m▱\e[0m \e[31m90%\e[0m \e[2m·\e[0m 180.0k
"full"\t\e[1mfull\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;188;226;173m▰\e[0m\e[38;2;228;226;174m▰\e[0m\e[38;2;248;209;174m▰\e[0m\e[38;2;246;174;171m▰\e[0m\e[38;2;243;139;168m▰\e[0m \e[31m100%\e[0m \e[2m·\e[0m 200.0k
"over"\t\e[1mover\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;188;226;173m▰\e[0m\e[38;2;228;226;174m▰\e[0m\e[38;2;248;209;174m▰\e[0m\e[38;2;246;174;171m▰\e[0m\e[38;2;243;139;168m▰\e[0m \e[31m125%\e[0m \e[2m·\e[0m 250.0k
"zero"\t\e[1mzero\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;74;74;74m▱▱▱▱▱▱\e[0m \e[32m0%\e[0m \e[2m·\e[0m 0`,
		},
		{
			name:    `The gradient theme with NO_COLOR keeps the block characters but drops every color in agent rows`,
			session: `{"columns":200,"tasks":[{"id":"ok","name":"ok","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":99999},{"id":"warn","name":"warn","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":100000},{"id":"bad","name":"bad","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":180000},{"id":"full","name":"full","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":200000},{"id":"over","name":"over","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":250000},{"id":"zero","name":"zero","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":0}]}`,
			config:  `theme: gradient`,
			env:     map[string]string{`NO_COLOR`: `1`},
			want: `"ok"\tok · opus-4-7 · ctx: ▰▰▰▰▰▱▱▱▱▱ 49% · 99.9k
"warn"\twarn · opus-4-7 · ctx: ▰▰▰▰▰▱▱▱▱▱ 50% · 100.0k
"bad"\tbad · opus-4-7 · ctx: ▰▰▰▰▰▰▰▰▰▱ 90% · 180.0k
"full"\tfull · opus-4-7 · ctx: ▰▰▰▰▰▰▰▰▰▰ 100% · 200.0k
"over"\tover · opus-4-7 · ctx: ▰▰▰▰▰▰▰▰▰▰ 125% · 250.0k
"zero"\tzero · opus-4-7 · ctx: ▱▱▱▱▱▱▱▱▱▱ 0% · 0`,
		},
		{
			name:    `The gradient theme's context bars either side of the warning and bad thresholds`,
			session: `{"columns":200,"tasks":[{"id":"ok","name":"ok","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":99999},{"id":"warn","name":"warn","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":100000},{"id":"bad","name":"bad","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":180000},{"id":"full","name":"full","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":200000},{"id":"over","name":"over","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":250000},{"id":"zero","name":"zero","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":0}]}`,
			config:  `theme: gradient`,
			want: `"ok"\t\e[1mok\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;237;226;174m▰\e[0m\e[38;2;74;74;74m▱▱▱▱▱\e[0m \e[32m49%\e[0m \e[2m·\e[0m 99.9k
"warn"\t\e[1mwarn\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;237;226;174m▰\e[0m\e[38;2;74;74;74m▱▱▱▱▱\e[0m \e[33m50%\e[0m \e[2m·\e[0m 100.0k
"bad"\t\e[1mbad\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;237;226;174m▰\e[0m\e[38;2;249;217;175m▰\e[0m\e[38;2;248;198;173m▰\e[0m\e[38;2;246;178;172m▰\e[0m\e[38;2;245;159;170m▰\e[0m\e[38;2;74;74;74m▱\e[0m \e[31m90%\e[0m \e[2m·\e[0m 180.0k
"full"\t\e[1mfull\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;237;226;174m▰\e[0m\e[38;2;249;217;175m▰\e[0m\e[38;2;248;198;173m▰\e[0m\e[38;2;246;178;172m▰\e[0m\e[38;2;245;159;170m▰\e[0m\e[38;2;243;139;168m▰\e[0m \e[31m100%\e[0m \e[2m·\e[0m 200.0k
"over"\t\e[1mover\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;237;226;174m▰\e[0m\e[38;2;249;217;175m▰\e[0m\e[38;2;248;198;173m▰\e[0m\e[38;2;246;178;172m▰\e[0m\e[38;2;245;159;170m▰\e[0m\e[38;2;243;139;168m▰\e[0m \e[31m125%\e[0m \e[2m·\e[0m 250.0k
"zero"\t\e[1mzero\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;74;74;74m▱▱▱▱▱▱▱▱▱▱\e[0m \e[32m0%\e[0m \e[2m·\e[0m 0`,
		},
		{
			name:    `Invalid JSON input prints no rows`,
			session: `{"tasks":[`,
			want:    ``,
		},
		{
			name:    `A long description is shown in full, not cut to fit the panel width`,
			session: `{"columns":200,"tasks":[{"id":"t1","name":"writer","type":"local_agent","description":"Investigate why the nightly reconciliation job reports a mismatch between the ledger and the settlement files for accounts opened before the migration, then propose and implement a fix with tests","model":"claude-sonnet-4-5","effort":"medium","contextWindowSize":200000,"tokenCount":123456}]}`,
			want:    `"t1"\t\e[1mwriter\e[0m \e[2m·\e[0m sonnet-4-5\e[2m [medium]\e[0m \e[2m·\e[0m ctx: \e[33m██████▏\e[0m\e[2m░░░\e[0m \e[33m61%\e[0m \e[2m·\e[0m 123.4k \e[2m·\e[0m \e[2mInvestigate why the nightly reconciliation job reports a mismatch between the ledger and the settlement files for accounts opened before the migration, then propose and implement a fix with tests\e[0m`,
		},
		{
			name:    `Tasks from an older Claude Code without model or effort fields fall back to their type and token count`,
			session: `{"tasks":[{"id":"t1","type":"local_agent","status":"running","description":"Older Claude Code without model fields","tokenCount":2500},{"id":"t2","name":"named","type":"local_agent","description":"Also old","tokenCount":1000000}]}`,
			want: `"t1"\t\e[1mlocal_agent\e[0m \e[2m·\e[0m 2.5k \e[2m·\e[0m \e[2mOlder Claude Code without model fields\e[0m
"t2"\t\e[1mnamed\e[0m \e[2m·\e[0m 1.0M \e[2m·\e[0m \e[2mAlso old\e[0m`,
		},
		{
			name:    `Rows omit the token count or the context bar when the data for them is missing or zero`,
			session: `{"tasks":[{"id":"t1","name":"a","type":"local_agent","description":"No token count","model":"claude-opus-4-7","effort":"high"},{"id":"t2","name":"b","type":"local_agent","model":"claude-opus-4-7","tokenCount":4200},{"id":"t3","name":"c","type":"local_agent","description":"Zero window","model":"claude-opus-4-7","contextWindowSize":0,"tokenCount":4200}]}`,
			want: `"t1"\t\e[1ma\e[0m \e[2m·\e[0m opus-4-7\e[2m [high]\e[0m \e[2m·\e[0m \e[2mNo token count\e[0m
"t2"\t\e[1mb\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m 4.2k
"t3"\t\e[1mc\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m 4.2k \e[2m·\e[0m \e[2mZero window\e[0m`,
		},
		{
			name:    `A narrow 80 column panel uses the narrow context bar`,
			session: `{"columns":80,"tasks":[{"id":"t1","name":"reviewer","description":"Narrow panel","model":"claude-sonnet-4-5","contextWindowSize":200000,"tokenCount":150000}]}`,
			want:    `"t1"\t\e[1mreviewer\e[0m \e[2m·\e[0m sonnet-4-5 \e[2m·\e[0m ctx: \e[33m████▌\e[0m\e[2m░\e[0m \e[33m75%\e[0m \e[2m·\e[0m 150.0k \e[2m·\e[0m \e[2mNarrow panel\e[0m`,
		},
		{
			name:    `NO_COLOR removes every color from agent rows`,
			session: `{"tasks":[{"id":"t1","name":"reviewer","type":"local_agent","description":"Plain text only","model":"claude-sonnet-4-5","effort":"high","contextWindowSize":200000,"tokenCount":150000}]}`,
			env:     map[string]string{`NO_COLOR`: `1`},
			want:    `"t1"\treviewer · sonnet-4-5 [high] · ctx: ███████▌░░ 75% · 150.0k · Plain text only`,
		},
		{
			name:    `A payload with no tasks key prints no rows`,
			session: `{"session_id":"s"}`,
			want:    ``,
		},
		{
			name:    `A numeric effort is shown as a number and a model without a claude- prefix is shown as it is`,
			session: `{"tasks":[{"id":"t1","name":"x","model":"custom-model-1","effort":31999,"contextWindowSize":100000,"tokenCount":33333,"description":"numeric effort"}]}`,
			want:    `"t1"\t\e[1mx\e[0m \e[2m·\e[0m custom-model-1\e[2m [31999]\e[0m \e[2m·\e[0m ctx: \e[32m███▎\e[0m\e[2m░░░░░░\e[0m \e[32m33%\e[0m \e[2m·\e[0m 33.3k \e[2m·\e[0m \e[2mnumeric effort\e[0m`,
		},
		{
			name:    `A numeric task id is echoed back as a number`,
			session: `{"tasks":[{"id":42,"name":"n","description":"numeric id echoed as is"}]}`,
			want:    `42\t\e[1mn\e[0m \e[2m·\e[0m \e[2mnumeric id echoed as is\e[0m`,
		},
		{
			name:    `Tasks without an id, or with a null id, get no row so Claude Code uses its default`,
			session: `{"tasks":[{"name":"anonymous","description":"no id so default rendering"},{"id":"kept","name":"kept","description":"has id"},{"id":null,"name":"nullid"}]}`,
			want:    `"kept"\t\e[1mkept\e[0m \e[2m·\e[0m \e[2mhas id\e[0m`,
		},
		{
			name:    `Context bars either side of the warning and bad thresholds`,
			session: `{"columns":200,"tasks":[{"id":"ok","name":"ok","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":99999},{"id":"warn","name":"warn","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":100000},{"id":"bad","name":"bad","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":180000},{"id":"full","name":"full","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":200000},{"id":"over","name":"over","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":250000},{"id":"zero","name":"zero","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":0}]}`,
			want: `"ok"\t\e[1mok\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[32m████▉\e[0m\e[2m░░░░░\e[0m \e[32m49%\e[0m \e[2m·\e[0m 99.9k
"warn"\t\e[1mwarn\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[33m█████\e[0m\e[2m░░░░░\e[0m \e[33m50%\e[0m \e[2m·\e[0m 100.0k
"bad"\t\e[1mbad\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[31m█████████\e[0m\e[2m░\e[0m \e[31m90%\e[0m \e[2m·\e[0m 180.0k
"full"\t\e[1mfull\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[31m██████████\e[0m \e[31m100%\e[0m \e[2m·\e[0m 200.0k
"over"\t\e[1mover\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[31m██████████\e[0m \e[31m125%\e[0m \e[2m·\e[0m 250.0k
"zero"\t\e[1mzero\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[2m░░░░░░░░░░\e[0m \e[32m0%\e[0m \e[2m·\e[0m 0`,
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.rows(t))
		})
	}
}
