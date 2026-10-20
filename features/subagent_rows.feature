@subagent_rows
Feature: Subagent rows
  The subagent command prints one row for each agent in Claude Code's agent panel. Each expected row
  is the task id, a tab written as \t, then the row content.

  Scenario: A single running agent with a model, effort, context bar, token count and description
    Given the session:
      """json
      {
        "session_id": "s1",
        "columns": 200,
        "tasks": [
          {
            "id": "task_1",
            "name": "reviewer",
            "type": "local_agent",
            "agentType": "code-reviewer",
            "status": "running",
            "description": "Review the payment retry change",
            "label": "Reading diff",
            "startTime": 1760000000000,
            "model": "claude-sonnet-4-5-20250929",
            "effort": "high",
            "contextWindowSize": 200000,
            "tokenCount": 84500,
            "tokenSamples": [
              80000,
              84500
            ],
            "cwd": "/tmp/work"
          }
        ]
      }
      """
    When the subagent rows render
    Then the output is:
      """
      "task_1"\treviewer · Sonnet 4.5 [high] · ctx: ████▎░░░░░ 42% · 84.5k · Review the payment retry change
                BBBBBBBB d           ddddddd d      gggggddddd ggg d       d ddddddddddddddddddddddddddddddd
      """

  Scenario: Several agents of different models, efforts and sizes, one row each
    Given the session:
      """json
      {
        "columns": 200,
        "tasks": [
          {
            "id": "a",
            "name": "explorer",
            "type": "local_agent",
            "description": "Map the config package",
            "model": "claude-haiku-4-5-20251001",
            "effort": "low",
            "contextWindowSize": 200000,
            "tokenCount": 12000
          },
          {
            "id": "b",
            "type": "local_agent",
            "description": "Draft the migration",
            "model": "claude-opus-4-7",
            "effort": "xhigh",
            "contextWindowSize": 1000000,
            "tokenCount": 512345
          },
          {
            "id": "c",
            "name": "tester",
            "type": "local_agent",
            "description": "Run the suite",
            "model": "claude-fable-5",
            "contextWindowSize": 200000,
            "tokenCount": 185000
          },
          {
            "id": "d",
            "name": "tiny",
            "type": "local_agent",
            "description": "Just started",
            "model": "claude-sonnet-4-5",
            "contextWindowSize": 200000,
            "tokenCount": 950
          }
        ]
      }
      """
    When the subagent rows render
    Then the output is:
      """
      "a"\texplorer · Haiku 4.5 [low] · ctx: ▋░░░░░░░░░ 6% · 12.0k · Map the config package
           BBBBBBBB d          dddddd d      gddddddddd gg d       d dddddddddddddddddddddd
      "b"\tlocal_agent · Opus 4.7 [xhigh] · ctx: █████▏░░░░ 51% · 512.3k · Draft the migration
           BBBBBBBBBBB d         dddddddd d      yyyyyydddd yyy d        d ddddddddddddddddddd
      "c"\ttester · Fable 5 · ctx: █████████▎ 92% · 185.0k · Run the suite
           BBBBBB d         d      rrrrrrrrrr rrr d        d ddddddddddddd
      "d"\ttiny · Sonnet 4.5 · ctx: ░░░░░░░░░░ 0% · 950 · Just started
           BBBB d            d      dddddddddd gg d     d dddddddddddd
      """
