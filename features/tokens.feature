@tokens
Feature: Token counts
  The tokens segment shows the input and output token counts, abbreviated with k and M.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: Token counts in the thousands are shown with a k suffix
    Given the session:
      """json
      {
        "context_window": {
          "total_input_tokens": 1500,
          "total_output_tokens": 999999
        }
      }
      """
    And the config:
      """
      segments = tokens
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the text is:
      """
      ↓ 1.5k · ↑ 999.9k
      """
