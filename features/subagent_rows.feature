@subagent_rows
Feature: Subagent rows
  The subagent command prints one row for each agent in Claude Code's agent panel. Each expected row
  is the task id, a tab written as \t, then the row content.

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
