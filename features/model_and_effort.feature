@model_and_effort
Feature: Model and effort
  The model segment shows the display name, followed by the effort level when there is one.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: A payload with only a model name shows just that
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"session_id":"abc123"}
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7
      """

  Scenario: A payload without an effort level shows just the model name
    Given the session:
      """json
      {"model":{"display_name":"Sonnet 5.5"},"session_id":"abc123"}
      """
    When the status line renders
    Then the output is:
      """
      Sonnet 5.5
      """

  Scenario: model.effort = false hides the effort level after the model name
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}
      """
    And the config:
      """
      model.effort = false
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7
      """
