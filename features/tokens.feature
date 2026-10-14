@tokens
Feature: Token counts
  The tokens segment shows the input and output token counts, abbreviated with k and M.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario Outline: Token counts <rule>
    Given the session:
      """json
      {"context_window":{"total_input_tokens":<in>,"total_output_tokens":<out>}}
      """
    And the config:
      """
      segments = tokens
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the output is:
      """
      <output>
      """

    Examples:
      | in      | out     | rule                                                                        | output            |
      | 999     | 1000    | below a thousand are shown as they are, and exactly a thousand becomes 1.0k | ↓ 999 · ↑ 1.0k    |
      | 1500    | 999999  | in the thousands are shown with a k suffix                                  | ↓ 1.5k · ↑ 999.9k |
      | 1000000 | 1500000 | in the millions are shown with an M suffix                                  | ↓ 1.0M · ↑ 1.5M   |

  Scenario: A payload with only an output token count still shows the tokens segment
    Given the session:
      """json
      {"context_window":{"total_output_tokens":0}}
      """
    And the config:
      """
      segments = tokens
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the output is:
      """
      ↑ 0
      """

  Scenario: The tokens segment with colour on, with a dimmed separator between the input and output counts
    Given the session:
      """json
      {"context_window":{"total_input_tokens":1500,"total_output_tokens":250}}
      """
    And the config:
      """
      segments = tokens
      """
    When the status line renders
    Then the output is:
      """
      ↓ 1.5k \e[2m·\e[0m ↑ 250
      """
