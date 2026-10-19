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
