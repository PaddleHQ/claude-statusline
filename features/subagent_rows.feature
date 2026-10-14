@subagent_rows
Feature: Subagent rows
  The subagent command prints one row for each agent in Claude Code's agent panel. Each expected row
  is the task id, a tab written as \t, then the row content.

  Scenario: Control characters and escape sequences in names, descriptions, models and efforts are stripped from the rows
    Given the session:
      """json
      {"tasks":[{"id":"t1","name":"bad\u001b[31mname","type":"local_agent","description":"line one\nline two \u001b]8;;http://evil\u0007link\u009b31m end\u0000","model":"claude-sonnet-4-5\u001b[0m","effort":"hi\u001bgh","contextWindowSize":200000,"tokenCount":50000}]}
      """
    When the subagent rows render
    Then the output is:
      """
      "t1"\t\e[1mbad[31mname\e[0m \e[2m·\e[0m sonnet-4-5[0m\e[2m [high]\e[0m \e[2m·\e[0m ctx: \e[32m██▌\e[0m\e[2m░░░░░░░\e[0m \e[32m25%\e[0m \e[2m·\e[0m 50.0k \e[2m·\e[0m \e[2mline oneline two ]8;;http://evillink31m end\e[0m
      """

  Scenario: The ctx.width, ctx.warn, ctx.bad and separator settings apply to agent rows
    Given the session:
      """json
      {"tasks":[{"id":"t1","name":"reviewer","description":"Custom thresholds","model":"claude-sonnet-4-5","contextWindowSize":200000,"tokenCount":60000}]}
      """
    And the config:
      """
      ctx.width = 5
      ctx.warn = 20
      ctx.bad = 25
      separator = |
      """
    When the subagent rows render
    Then the output is:
      """
      "t1"\t\e[1mreviewer\e[0m \e[2m|\e[0m sonnet-4-5 \e[2m|\e[0m ctx: \e[31m█▌\e[0m\e[2m░░░\e[0m \e[31m30%\e[0m \e[2m|\e[0m 60.0k \e[2m|\e[0m \e[2mCustom thresholds\e[0m
      """

  Scenario: Empty input prints no rows
    Given the session:
      """json
      """
    When the subagent rows render
    Then the output is:
      """
      """

  Scenario: An empty task list prints no rows
    Given the session:
      """json
      {"tasks":[]}
      """
    When the subagent rows render
    Then the output is:
      """
      """

  Scenario: The gradient theme's context bars in a 100 column panel, at each colour threshold
    Given the session:
      """json
      {"columns":100,"tasks":[{"id":"ok","name":"ok","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":99999},{"id":"warn","name":"warn","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":100000},{"id":"bad","name":"bad","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":180000},{"id":"full","name":"full","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":200000},{"id":"over","name":"over","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":250000},{"id":"zero","name":"zero","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":0}]}
      """
    And the config:
      """
      theme = gradient
      """
    When the subagent rows render
    Then the output is:
      """
      "ok"\t\e[1mok\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;188;226;173m▰\e[0m\e[38;2;228;226;174m▰\e[0m\e[38;2;74;74;74m▱▱▱\e[0m \e[32m49%\e[0m \e[2m·\e[0m 99.9k
      "warn"\t\e[1mwarn\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;188;226;173m▰\e[0m\e[38;2;228;226;174m▰\e[0m\e[38;2;74;74;74m▱▱▱\e[0m \e[33m50%\e[0m \e[2m·\e[0m 100.0k
      "bad"\t\e[1mbad\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;188;226;173m▰\e[0m\e[38;2;228;226;174m▰\e[0m\e[38;2;248;209;174m▰\e[0m\e[38;2;246;174;171m▰\e[0m\e[38;2;74;74;74m▱\e[0m \e[31m90%\e[0m \e[2m·\e[0m 180.0k
      "full"\t\e[1mfull\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;188;226;173m▰\e[0m\e[38;2;228;226;174m▰\e[0m\e[38;2;248;209;174m▰\e[0m\e[38;2;246;174;171m▰\e[0m\e[38;2;243;139;168m▰\e[0m \e[31m100%\e[0m \e[2m·\e[0m 200.0k
      "over"\t\e[1mover\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;188;226;173m▰\e[0m\e[38;2;228;226;174m▰\e[0m\e[38;2;248;209;174m▰\e[0m\e[38;2;246;174;171m▰\e[0m\e[38;2;243;139;168m▰\e[0m \e[31m125%\e[0m \e[2m·\e[0m 250.0k
      "zero"\t\e[1mzero\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;74;74;74m▱▱▱▱▱▱\e[0m \e[32m0%\e[0m \e[2m·\e[0m 0
      """

  Scenario: The gradient theme with NO_COLOR keeps the block characters but drops every colour in agent rows
    Given the session:
      """json
      {"columns":200,"tasks":[{"id":"ok","name":"ok","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":99999},{"id":"warn","name":"warn","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":100000},{"id":"bad","name":"bad","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":180000},{"id":"full","name":"full","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":200000},{"id":"over","name":"over","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":250000},{"id":"zero","name":"zero","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":0}]}
      """
    And the config:
      """
      theme = gradient
      """
    And the environment variable "NO_COLOR" is "1"
    When the subagent rows render
    Then the output is:
      """
      "ok"\tok · opus-4-7 · ctx: ▰▰▰▰▰▱▱▱▱▱ 49% · 99.9k
      "warn"\twarn · opus-4-7 · ctx: ▰▰▰▰▰▱▱▱▱▱ 50% · 100.0k
      "bad"\tbad · opus-4-7 · ctx: ▰▰▰▰▰▰▰▰▰▱ 90% · 180.0k
      "full"\tfull · opus-4-7 · ctx: ▰▰▰▰▰▰▰▰▰▰ 100% · 200.0k
      "over"\tover · opus-4-7 · ctx: ▰▰▰▰▰▰▰▰▰▰ 125% · 250.0k
      "zero"\tzero · opus-4-7 · ctx: ▱▱▱▱▱▱▱▱▱▱ 0% · 0
      """

  Scenario: The gradient theme's context bars either side of the warning and bad thresholds
    Given the session:
      """json
      {"columns":200,"tasks":[{"id":"ok","name":"ok","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":99999},{"id":"warn","name":"warn","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":100000},{"id":"bad","name":"bad","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":180000},{"id":"full","name":"full","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":200000},{"id":"over","name":"over","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":250000},{"id":"zero","name":"zero","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":0}]}
      """
    And the config:
      """
      theme = gradient
      """
    When the subagent rows render
    Then the output is:
      """
      "ok"\t\e[1mok\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;237;226;174m▰\e[0m\e[38;2;74;74;74m▱▱▱▱▱\e[0m \e[32m49%\e[0m \e[2m·\e[0m 99.9k
      "warn"\t\e[1mwarn\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;237;226;174m▰\e[0m\e[38;2;74;74;74m▱▱▱▱▱\e[0m \e[33m50%\e[0m \e[2m·\e[0m 100.0k
      "bad"\t\e[1mbad\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;237;226;174m▰\e[0m\e[38;2;249;217;175m▰\e[0m\e[38;2;248;198;173m▰\e[0m\e[38;2;246;178;172m▰\e[0m\e[38;2;245;159;170m▰\e[0m\e[38;2;74;74;74m▱\e[0m \e[31m90%\e[0m \e[2m·\e[0m 180.0k
      "full"\t\e[1mfull\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;237;226;174m▰\e[0m\e[38;2;249;217;175m▰\e[0m\e[38;2;248;198;173m▰\e[0m\e[38;2;246;178;172m▰\e[0m\e[38;2;245;159;170m▰\e[0m\e[38;2;243;139;168m▰\e[0m \e[31m100%\e[0m \e[2m·\e[0m 200.0k
      "over"\t\e[1mover\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;237;226;174m▰\e[0m\e[38;2;249;217;175m▰\e[0m\e[38;2;248;198;173m▰\e[0m\e[38;2;246;178;172m▰\e[0m\e[38;2;245;159;170m▰\e[0m\e[38;2;243;139;168m▰\e[0m \e[31m125%\e[0m \e[2m·\e[0m 250.0k
      "zero"\t\e[1mzero\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[38;2;74;74;74m▱▱▱▱▱▱▱▱▱▱\e[0m \e[32m0%\e[0m \e[2m·\e[0m 0
      """

  Scenario: Invalid JSON input prints no rows
    Given the session:
      """json
      {"tasks":[
      """
    When the subagent rows render
    Then the output is:
      """
      """

  Scenario: A long description is shown in full, not cut to fit the panel width
    Given the session:
      """json
      {"columns":200,"tasks":[{"id":"t1","name":"writer","type":"local_agent","description":"Investigate why the nightly reconciliation job reports a mismatch between the ledger and the settlement files for accounts opened before the migration, then propose and implement a fix with tests","model":"claude-sonnet-4-5","effort":"medium","contextWindowSize":200000,"tokenCount":123456}]}
      """
    When the subagent rows render
    Then the output is:
      """
      "t1"\t\e[1mwriter\e[0m \e[2m·\e[0m sonnet-4-5\e[2m [medium]\e[0m \e[2m·\e[0m ctx: \e[33m██████▏\e[0m\e[2m░░░\e[0m \e[33m61%\e[0m \e[2m·\e[0m 123.4k \e[2m·\e[0m \e[2mInvestigate why the nightly reconciliation job reports a mismatch between the ledger and the settlement files for accounts opened before the migration, then propose and implement a fix with tests\e[0m
      """

  Scenario: Tasks from an older Claude Code without model or effort fields fall back to their type and token count
    Given the session:
      """json
      {"tasks":[{"id":"t1","type":"local_agent","status":"running","description":"Older Claude Code without model fields","tokenCount":2500},{"id":"t2","name":"named","type":"local_agent","description":"Also old","tokenCount":1000000}]}
      """
    When the subagent rows render
    Then the output is:
      """
      "t1"\t\e[1mlocal_agent\e[0m \e[2m·\e[0m 2.5k \e[2m·\e[0m \e[2mOlder Claude Code without model fields\e[0m
      "t2"\t\e[1mnamed\e[0m \e[2m·\e[0m 1.0M \e[2m·\e[0m \e[2mAlso old\e[0m
      """

  Scenario: Rows omit the token count or the context bar when the data for them is missing or zero
    Given the session:
      """json
      {"tasks":[{"id":"t1","name":"a","type":"local_agent","description":"No token count","model":"claude-opus-4-7","effort":"high"},{"id":"t2","name":"b","type":"local_agent","model":"claude-opus-4-7","tokenCount":4200},{"id":"t3","name":"c","type":"local_agent","description":"Zero window","model":"claude-opus-4-7","contextWindowSize":0,"tokenCount":4200}]}
      """
    When the subagent rows render
    Then the output is:
      """
      "t1"\t\e[1ma\e[0m \e[2m·\e[0m opus-4-7\e[2m [high]\e[0m \e[2m·\e[0m \e[2mNo token count\e[0m
      "t2"\t\e[1mb\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m 4.2k
      "t3"\t\e[1mc\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m 4.2k \e[2m·\e[0m \e[2mZero window\e[0m
      """

  Scenario: A narrow 80 column panel uses the narrow context bar
    Given the session:
      """json
      {"columns":80,"tasks":[{"id":"t1","name":"reviewer","description":"Narrow panel","model":"claude-sonnet-4-5","contextWindowSize":200000,"tokenCount":150000}]}
      """
    When the subagent rows render
    Then the output is:
      """
      "t1"\t\e[1mreviewer\e[0m \e[2m·\e[0m sonnet-4-5 \e[2m·\e[0m ctx: \e[33m████▌\e[0m\e[2m░\e[0m \e[33m75%\e[0m \e[2m·\e[0m 150.0k \e[2m·\e[0m \e[2mNarrow panel\e[0m
      """

  Scenario: NO_COLOR removes every colour from agent rows
    Given the session:
      """json
      {"tasks":[{"id":"t1","name":"reviewer","type":"local_agent","description":"Plain text only","model":"claude-sonnet-4-5","effort":"high","contextWindowSize":200000,"tokenCount":150000}]}
      """
    And the environment variable "NO_COLOR" is "1"
    When the subagent rows render
    Then the output is:
      """
      "t1"\treviewer · sonnet-4-5 [high] · ctx: ███████▌░░ 75% · 150.0k · Plain text only
      """

  Scenario: A payload with no tasks key prints no rows
    Given the session:
      """json
      {"session_id":"s"}
      """
    When the subagent rows render
    Then the output is:
      """
      """

  Scenario: A numeric effort is shown as a number and a model without a claude- prefix is shown as it is
    Given the session:
      """json
      {"tasks":[{"id":"t1","name":"x","model":"custom-model-1","effort":31999,"contextWindowSize":100000,"tokenCount":33333,"description":"numeric effort"}]}
      """
    When the subagent rows render
    Then the output is:
      """
      "t1"\t\e[1mx\e[0m \e[2m·\e[0m custom-model-1\e[2m [31999]\e[0m \e[2m·\e[0m ctx: \e[32m███▎\e[0m\e[2m░░░░░░\e[0m \e[32m33%\e[0m \e[2m·\e[0m 33.3k \e[2m·\e[0m \e[2mnumeric effort\e[0m
      """

  Scenario: A numeric task id is echoed back as a number
    Given the session:
      """json
      {"tasks":[{"id":42,"name":"n","description":"numeric id echoed as is"}]}
      """
    When the subagent rows render
    Then the output is:
      """
      42\t\e[1mn\e[0m \e[2m·\e[0m \e[2mnumeric id echoed as is\e[0m
      """

  Scenario: A single running agent with a model, effort, context bar, token count and description
    Given the session:
      """json
      {"session_id":"s1","columns":200,"tasks":[{"id":"task_1","name":"reviewer","type":"local_agent","agentType":"code-reviewer","status":"running","description":"Review the payment retry change","label":"Reading diff","startTime":1760000000000,"model":"claude-sonnet-4-5-20250929","effort":"high","contextWindowSize":200000,"tokenCount":84500,"tokenSamples":[80000,84500],"cwd":"/tmp/work"}]}
      """
    When the subagent rows render
    Then the output is:
      """
      "task_1"\t\e[1mreviewer\e[0m \e[2m·\e[0m sonnet-4-5\e[2m [high]\e[0m \e[2m·\e[0m ctx: \e[32m████▎\e[0m\e[2m░░░░░\e[0m \e[32m42%\e[0m \e[2m·\e[0m 84.5k \e[2m·\e[0m \e[2mReview the payment retry change\e[0m
      """

  Scenario: Tasks without an id, or with a null id, get no row so Claude Code uses its default
    Given the session:
      """json
      {"tasks":[{"name":"anonymous","description":"no id so default rendering"},{"id":"kept","name":"kept","description":"has id"},{"id":null,"name":"nullid"}]}
      """
    When the subagent rows render
    Then the output is:
      """
      "kept"\t\e[1mkept\e[0m \e[2m·\e[0m \e[2mhas id\e[0m
      """

  Scenario: Several agents of different models, efforts and sizes, one row each
    Given the session:
      """json
      {"columns":200,"tasks":[{"id":"a","name":"explorer","type":"local_agent","description":"Map the config package","model":"claude-haiku-4-5-20251001","effort":"low","contextWindowSize":200000,"tokenCount":12000},{"id":"b","type":"local_agent","description":"Draft the migration","model":"claude-opus-4-7","effort":"xhigh","contextWindowSize":1000000,"tokenCount":512345},{"id":"c","name":"tester","type":"local_agent","description":"Run the suite","model":"claude-fable-5","contextWindowSize":200000,"tokenCount":185000},{"id":"d","name":"tiny","type":"local_agent","description":"Just started","model":"claude-sonnet-4-5","contextWindowSize":200000,"tokenCount":950}]}
      """
    When the subagent rows render
    Then the output is:
      """
      "a"\t\e[1mexplorer\e[0m \e[2m·\e[0m haiku-4-5\e[2m [low]\e[0m \e[2m·\e[0m ctx: \e[32m▋\e[0m\e[2m░░░░░░░░░\e[0m \e[32m6%\e[0m \e[2m·\e[0m 12.0k \e[2m·\e[0m \e[2mMap the config package\e[0m
      "b"\t\e[1mlocal_agent\e[0m \e[2m·\e[0m opus-4-7\e[2m [xhigh]\e[0m \e[2m·\e[0m ctx: \e[33m█████▏\e[0m\e[2m░░░░\e[0m \e[33m51%\e[0m \e[2m·\e[0m 512.3k \e[2m·\e[0m \e[2mDraft the migration\e[0m
      "c"\t\e[1mtester\e[0m \e[2m·\e[0m fable-5 \e[2m·\e[0m ctx: \e[31m█████████▎\e[0m \e[31m92%\e[0m \e[2m·\e[0m 185.0k \e[2m·\e[0m \e[2mRun the suite\e[0m
      "d"\t\e[1mtiny\e[0m \e[2m·\e[0m sonnet-4-5 \e[2m·\e[0m ctx: \e[2m░░░░░░░░░░\e[0m \e[32m0%\e[0m \e[2m·\e[0m 950 \e[2m·\e[0m \e[2mJust started\e[0m
      """

  Scenario: Context bars either side of the warning and bad thresholds
    Given the session:
      """json
      {"columns":200,"tasks":[{"id":"ok","name":"ok","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":99999},{"id":"warn","name":"warn","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":100000},{"id":"bad","name":"bad","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":180000},{"id":"full","name":"full","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":200000},{"id":"over","name":"over","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":250000},{"id":"zero","name":"zero","model":"claude-opus-4-7","contextWindowSize":200000,"tokenCount":0}]}
      """
    When the subagent rows render
    Then the output is:
      """
      "ok"\t\e[1mok\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[32m████▉\e[0m\e[2m░░░░░\e[0m \e[32m49%\e[0m \e[2m·\e[0m 99.9k
      "warn"\t\e[1mwarn\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[33m█████\e[0m\e[2m░░░░░\e[0m \e[33m50%\e[0m \e[2m·\e[0m 100.0k
      "bad"\t\e[1mbad\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[31m█████████\e[0m\e[2m░\e[0m \e[31m90%\e[0m \e[2m·\e[0m 180.0k
      "full"\t\e[1mfull\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[31m██████████\e[0m \e[31m100%\e[0m \e[2m·\e[0m 200.0k
      "over"\t\e[1mover\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[31m██████████\e[0m \e[31m125%\e[0m \e[2m·\e[0m 250.0k
      "zero"\t\e[1mzero\e[0m \e[2m·\e[0m opus-4-7 \e[2m·\e[0m ctx: \e[2m░░░░░░░░░░\e[0m \e[32m0%\e[0m \e[2m·\e[0m 0
      """
