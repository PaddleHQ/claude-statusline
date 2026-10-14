@context_bar
Feature: Context bar
  The ctx segment draws how full the context window is, in a colour that follows the warning and
  bad thresholds.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario Outline: Context use of <used>% is drawn in <colour>
    Given the session:
      """json
      {"context_window":{"used_percentage":<used>}}
      """
    And the config:
      """
      segments = ctx
      """
    When the status line renders
    Then the output is:
      """
      <output>
      """

    Examples:
      | used | colour | output                                   |
      | 30   | green  | ctx: \e[32m███\e[0m\e[2m░░░░░░░\e[0m 30% |
      | 50   | yellow | ctx: \e[33m█████\e[0m\e[2m░░░░░\e[0m 50% |
      | 90   | red    | ctx: \e[31m█████████\e[0m\e[2m░\e[0m 90% |

  Scenario: At 100 columns the context bar is narrower and a fractional percentage is rounded
    Given the session:
      """json
      {"context_window":{"used_percentage":64.6}}
      """
    And the config:
      """
      segments = ctx
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders at 100 columns
    Then the output is:
      """
      ctx: ███▉░░ 65%
      """

  Scenario: The ctx segment is hidden when the payload has no used percentage
    Given the session:
      """json
      {"context_window":{"total_input_tokens":5}}
      """
    And the config:
      """
      segments = ctx
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the output is:
      """
      """
