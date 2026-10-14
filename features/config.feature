@config
Feature: Configuration
  The config file is forgiving: junk, unknown keys and invalid values are ignored, and its values
  are shown as text and never executed.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: Config parsing skips comments, blank lines, junk and unknown keys, ignores spacing around =, and lets the last duplicate win
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}
      """
    And the config:
      """
      # a comment

         separator   =   |
      segments=model model
      model.effort = true
      model.effort = false
      model.effort = true
      model.label = ignored
      not a setting line
      unknown.key = 1
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2m|\e[0m Opus 4.7 [high]
      """

  Scenario: Malformed config keys are ignored while the valid ones around them still apply
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}
      """
    And the config:
      """
      model.effort.x = false
      Model.effort = false
      model-effort = false
      model..effort = false
      .effort = false
      model. = false
      segments.x = nothing
      separator = ok
      segments = model model
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2mok\e[0m Opus 4.7 [high]
      """

  Scenario: Invalid or empty config values fall back to the defaults
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}
      """
    And the config:
      """
      model.effort = maybe
      separator =
      segments = model model
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2m·\e[0m Opus 4.7 [high]
      """

  Scenario: The defaults apply when there is no config file
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high]
      """

  Scenario: An unknown name in the segments list is dropped and the rest are shown
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}
      """
    And the config:
      """
      segments = bogus model
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high]
      """

  Scenario: Unknown, malformed and path-like segment names are dropped, and an unknown theme falls back to the default
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}
      """
    And the config:
      """
      segments = model bogus Bad-Name ../x model
      theme = nonexistent
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2m·\e[0m Opus 4.7 [high]
      """

  Scenario: Shell metacharacters and command substitutions in config values are shown as text and never executed
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}
      """
    And the config:
      """
      segments = model model
      separator = "; touch @TMP@/pwned; " `touch @TMP@/pwned`
      unused.key = "; touch @TMP@/pwned; " `touch @TMP@/pwned` $(touch @TMP@/pwned)
      model.effort = $(touch @TMP@/pwned)\
      """
    When the status line renders at 999 columns
    Then the output is:
      """
      Opus 4.7 [high] \e[2m"; touch @TMP@/pwned; " `touch @TMP@/pwned`\e[0m Opus 4.7 [high]
      """

  Scenario: A command substitution in the separator setting is shown as text and never executed
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}
      """
    And the config:
      """
      segments = model model
      separator = $(touch @TMP@/pwned)
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2m$(touch @TMP@/pwned)\e[0m Opus 4.7 [high]
      """

  Scenario: Injection attempts in width and label settings are shown as text and never executed
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}
      """
    And the config:
      """
      rate5h.width = 5; touch @TMP@/pwned
      rate7d.label = `touch @TMP@/pwned`
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m `touch @TMP@/pwned`: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """
