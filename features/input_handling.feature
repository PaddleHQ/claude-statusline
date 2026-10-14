@input_handling
Feature: Input handling
  Bad input prints nothing rather than an error.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: Invalid JSON input prints nothing
    Given the session:
      """json
      {"model":
      """
    When the status line renders
    Then the text is:
      """
      """
