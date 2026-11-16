@context_bar
Feature: Context bar
  The ctx segment draws how full the context window is, in a colour that follows the warning and
  bad thresholds.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: Context use of 90% is drawn in red
    Given the session:
      """json
      {
        "context_window": {
          "used_percentage": 90
        }
      }
      """
    And the config:
      """
      segments: [ctx]
      """
    When the status line renders
    Then the output is:
      """
      ctx: █████████░ 90%
           rrrrrrrrrd
      """

  Scenario: A 1M window puts the compact marker at the 200k limit and stays quiet below it
    Given the session:
      """json
      {
        "context_window": {
          "used_percentage": 15,
          "context_window_size": 1000000,
          "current_usage": {
            "input_tokens": 0,
            "cache_creation_input_tokens": 0,
            "cache_read_input_tokens": 150000
          }
        }
      }
      """
    And the config:
      """
      segments: [ctx]
      """
    When the status line renders
    Then the output is:
      """
      ctx: █▌┃░░░░░░░ 15%
           ggBddddddd
      """

  Scenario: Reaching the compact limit turns the fill red and adds a warning
    Given the session:
      """json
      {
        "context_window": {
          "used_percentage": 21,
          "context_window_size": 1000000,
          "current_usage": {
            "input_tokens": 0,
            "cache_creation_input_tokens": 0,
            "cache_read_input_tokens": 210000
          }
        }
      }
      """
    And the config:
      """
      segments: [ctx]
      """
    When the status line renders
    Then the output is:
      """
      ctx: ██┃░░░░░░░ 21% ⚠ /compact
           rrBddddddd     rrrrrrrrrr
      """

  Scenario: A 200k window puts the compact marker at 80% of the window
    Given the session:
      """json
      {
        "context_window": {
          "used_percentage": 75,
          "context_window_size": 200000,
          "current_usage": {
            "input_tokens": 0,
            "cache_creation_input_tokens": 0,
            "cache_read_input_tokens": 150000
          }
        }
      }
      """
    And the config:
      """
      segments: [ctx]
      """
    When the status line renders
    Then the output is:
      """
      ctx: ███████▌┃░ 75%
           yyyyyyyyBd
      """

  Scenario: Passing 80% of a 200k window warns before the absolute limit
    Given the session:
      """json
      {
        "context_window": {
          "used_percentage": 82.5,
          "context_window_size": 200000,
          "current_usage": {
            "input_tokens": 0,
            "cache_creation_input_tokens": 0,
            "cache_read_input_tokens": 165000
          }
        }
      }
      """
    And the config:
      """
      segments: [ctx]
      """
    When the status line renders
    Then the output is:
      """
      ctx: ████████┃░ 82% ⚠ /compact
           rrrrrrrrBd     rrrrrrrrrr
      """

  Scenario: A missing window size is inferred from the used percentage
    Given the session:
      """json
      {
        "context_window": {
          "used_percentage": 21,
          "current_usage": {
            "input_tokens": 0,
            "cache_creation_input_tokens": 0,
            "cache_read_input_tokens": 210000
          }
        }
      }
      """
    And the config:
      """
      segments: [ctx]
      """
    When the status line renders
    Then the output is:
      """
      ctx: ██┃░░░░░░░ 21% ⚠ /compact
           rrBddddddd     rrrrrrrrrr
      """

  Scenario: With both compact limits off the bar has no marker or warning
    Given the session:
      """json
      {
        "context_window": {
          "used_percentage": 21,
          "context_window_size": 1000000,
          "current_usage": {
            "input_tokens": 0,
            "cache_creation_input_tokens": 0,
            "cache_read_input_tokens": 210000
          }
        }
      }
      """
    And the config:
      """
      segments: [ctx]
      ctx:
        compact_at: 0
        compact_pct: 0
      """
    When the status line renders
    Then the output is:
      """
      ctx: ██▏░░░░░░░ 21%
           gggddddddd
      """
