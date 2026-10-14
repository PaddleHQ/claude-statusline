@input_handling
Feature: Input handling
  Bad input prints nothing rather than an error, and control characters in the payload are stripped.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: Empty input prints nothing
    Given the session:
      """json
      """
    When the status line renders
    Then the output is:
      """
      """

  Scenario: Invalid JSON input prints nothing
    Given the session:
      """json
      {"model":
      """
    When the status line renders
    Then the output is:
      """
      """

  Scenario: Control characters in the model name from the payload are stripped before display
    Given the session:
      """json
      {"model":{"display_name":"Op\u001b[31mus"},"session_id":"abc123"}
      """
    When the status line renders
    Then the output is:
      """
      Op[31mus
      """
