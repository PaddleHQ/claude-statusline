@config
Feature: Configuration
  The config file is forgiving: an unknown segment name is dropped, segments_excluded leaves segments out
  without copying the list, and the values in it are shown as text and never executed.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: An unknown name in the segments list is dropped and the rest are shown
    Given the session:
      """json
      {
        "model": {
          "display_name": "Opus 4.7"
        },
        "effort": {
          "level": "high"
        }
      }
      """
    And the config:
      """
      segments: [bogus, model]
      """
    When the status line renders
    Then the text is:
      """
      Opus 4.7 [high]
      """

  Scenario: A segment named in segments_excluded is left out of the default list
    Given the session:
      """json
      {
        "model": {
          "display_name": "Opus 4.7"
        },
        "context_window": {
          "total_input_tokens": 1500,
          "total_output_tokens": 23400
        }
      }
      """
    And the config:
      """
      segments_excluded: [model]
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the text is:
      """
      ↓ 1.5k · ↑ 23.4k
      """

  Scenario: The cost segment, excluded by default, is shown once segments_excluded is emptied
    Given the session:
      """json
      {
        "model": {
          "display_name": "Opus 4.7"
        },
        "cost": {
          "total_cost_usd": 1.84
        }
      }
      """
    And the config:
      """
      segments_excluded: []
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the text is:
      """
      Opus 4.7 · $1.84
      """

  Scenario: Shell metacharacters and command substitutions in config values are shown as text and never executed
    Given the session:
      """json
      {
        "model": {
          "display_name": "Opus 4.7"
        },
        "effort": {
          "level": "high"
        }
      }
      """
    And the config:
      """
      segments: [model, model]
      separator: "\"; touch @TMP@/pwned; \" `touch @TMP@/pwned`"
      unused:
        key: "\"; touch @TMP@/pwned; \" `touch @TMP@/pwned` $(touch @TMP@/pwned)"
      model:
        effort: $(touch @TMP@/pwned)\
      """
    When the status line renders at 999 columns
    Then the output is:
      """
      Opus 4.7 [high] "; touch @TMP@/pwned; " `touch @TMP@/pwned` Opus 4.7 [high]
                      ddddddddddddddddddddddddddddddddddddddddddd
      """

  Scenario: Backslashes and quotes in a quoted config value are shown as text and never executed
    Given the session:
      """json
      {
        "model": {
          "display_name": "Opus 4.7"
        },
        "effort": {
          "level": "high"
        }
      }
      """
    And the config:
      """
      segments: [model, model]
      separator: '\$(touch @TMP@/pwned)\ "quoted" it''s'
      """
    When the status line renders at 999 columns
    Then the output is:
      """
      Opus 4.7 [high] \$(touch @TMP@/pwned)\ "quoted" it's Opus 4.7 [high]
                      dddddddddddddddddddddddddddddddddddd
      """
