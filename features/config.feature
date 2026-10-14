@config
Feature: Configuration
  The config file is forgiving: an unknown segment name is dropped, and the values in it are shown as
  text and never executed.

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
      segments = bogus model
      """
    When the status line renders
    Then the text is:
      """
      Opus 4.7 [high]
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
      segments = model model
      separator = "; touch @TMP@/pwned; " `touch @TMP@/pwned`
      unused.key = "; touch @TMP@/pwned; " `touch @TMP@/pwned` $(touch @TMP@/pwned)
      model.effort = $(touch @TMP@/pwned)\
      """
    When the status line renders at 999 columns
    Then the output is:
      """
      Opus 4.7 [high] "; touch @TMP@/pwned; " `touch @TMP@/pwned` Opus 4.7 [high]
                      ddddddddddddddddddddddddddddddddddddddddddd
      """
